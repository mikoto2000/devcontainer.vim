package devcontainer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const runCleanupTestDirEnv = "DEVCONTAINER_VIM_RUN_CLEANUP_TEST_DIR"
const runCleanupDockerDiagnostic = "fixture Docker daemon rejected the image"
const runCleanupProbeDiagnostic = "fixture Docker daemon could not exec the tool probe"

func installCleanupTestCommands(t *testing.T, dir string) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	cdr := filepath.Join(dir, "run-test-cdr"+suffix)
	for _, path := range []string{filepath.Join(dir, "docker"+suffix), cdr} {
		if err := os.WriteFile(path, binary, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(runCleanupTestDirEnv, dir)
	// Use the native CDR path even when the test runs under WSL.
	t.Setenv("WSL_INTEROP", "")
	t.Setenv("WSL_DISTRO_NAME", "")
	if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
		t.Fatal(err)
	}
	return cdr
}

func TestRunCleansUpAfterSetupFailure(t *testing.T) {
	for _, tt := range []struct {
		name     string
		commands []string
		exitCode int
		startCDR bool
		withTmux bool
	}{
		{name: "run", commands: []string{"run"}, exitCode: 13},
		{name: "uname", commands: []string{"run", "uname", "stop"}, exitCode: 17},
		{name: "probe-vim", commands: []string{"run", "uname", "probe-vim", "stop"}, exitCode: 1},
		{name: "probe-tmux", commands: []string{"run", "uname", "probe-vim", "probe-tmux", "stop"}, exitCode: 1, withTmux: true},
		{name: "cp", commands: []string{"run", "uname", "probe-vim", "cp", "stop"}, exitCode: 23, startCDR: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			cdr := installCleanupTestCommands(t, dir)
			// Keep an incorrect fallback offline and detect its attempted Docker copy.
			for _, name := range []string{"vim_amd64", "tmux_amd64"} {
				if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("DEVCONTAINER_VIM_RUN_CLEANUP_TEST_FAIL", tt.name)

			err := Run([]string{"test-image"}, !tt.startCDR, true, !tt.withTmux, cdr, dir, false, "", dir, "", nil)
			if tt.name == "run" {
				var startErr *ContainerStartError
				if !errors.As(err, &startErr) || !strings.Contains(err.Error(), runCleanupDockerDiagnostic) {
					t.Errorf("Run error = %v, want ContainerStartError with Docker stderr", err)
				}
			}
			if strings.HasPrefix(tt.name, "probe-") && (err == nil || !strings.Contains(err.Error(), runCleanupProbeDiagnostic)) {
				t.Errorf("Run error = %v, want the Docker tool probe diagnostic", err)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != tt.exitCode {
				t.Errorf("Run error = %v, want setup exit code %d", err, tt.exitCode)
			}

			commands, err := os.ReadFile(filepath.Join(dir, "commands"))
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Fields(string(commands)); !slices.Equal(got, tt.commands) {
				t.Errorf("Docker commands = %v, want %v", got, tt.commands)
			}
			for _, name := range []string{"container-running", "test-container"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("%s remains after Run: stat error = %v", name, err)
				}
			}
			if tt.startCDR {
				addr, err := os.ReadFile(filepath.Join(dir, "cdr-address"))
				if err != nil {
					t.Fatal(err)
				}
				deadline := time.Now().Add(time.Second)
				for {
					conn, err := net.DialTimeout("tcp", string(addr), 100*time.Millisecond)
					if err != nil {
						break
					}
					conn.Close()
					if time.Now().After(deadline) {
						t.Fatal("CDR is still accepting connections after Run")
					}
					time.Sleep(10 * time.Millisecond)
				}
			}
		})
	}
}

// Copies of the test executable act as Docker and CDR without either dependency.
func TestMain(m *testing.M) {
	if dir := os.Getenv(armVimTestDirEnv); dir != "" {
		switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
		case "docker", "test-arm-devcontainer":
			if err := runArmVimTestCommand(dir, os.Args[1:]); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(99)
			}
			os.Exit(0)
		}
	}
	if dir := os.Getenv(forwardTestDirEnv); dir != "" {
		switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
		case "docker", "test-forward-devcontainer":
			os.Exit(runForwardTestCommand(dir, os.Args))
		}
	}
	if dir := os.Getenv(stopTestDirEnv); dir != "" {
		switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
		case "docker", "test-devcontainer", "powershell":
			os.Exit(runStopTestCommand(dir, os.Args))
		}
	}
	if dir := os.Getenv(runCleanupTestDirEnv); dir != "" {
		switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
		case "docker":
			os.Exit(runCleanupTestDocker(dir, os.Args[1:]))
		case "run-test-cdr":
			if err := runCleanupTestCDR(dir); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(99)
			}
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

const runCleanupTestArchEnv = "DEVCONTAINER_VIM_RUN_CLEANUP_TEST_ARCH"

func runCleanupTestDocker(dir string, args []string) int {
	command := args[0]
	if command == "exec" && len(args) > 2 && args[1] == "test-container" {
		command = args[2]
		if command == "sh" && len(args) == 7 && args[3] == "-c" && args[4] == containerToolProbe && args[5] == "devcontainer-vim-tool-probe" {
			command = "probe-" + args[6]
		}
	}
	log, err := os.OpenFile(filepath.Join(dir, "commands"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return 99
	}
	_, err = fmt.Fprintln(log, command)
	log.Close()
	if err != nil {
		return 99
	}

	fail := os.Getenv("DEVCONTAINER_VIM_RUN_CLEANUP_TEST_FAIL")
	container := filepath.Join(dir, "container-running")
	switch command {
	case "run":
		if fail == "run" {
			fmt.Fprintln(os.Stderr, runCleanupDockerDiagnostic)
			return 13
		}
		if err := os.WriteFile(container, nil, 0600); err != nil {
			return 99
		}
		fmt.Print("  test-container\r\n")
		fmt.Fprintln(os.Stderr, "fixture Docker warning")
	case "uname":
		if fail == "uname" {
			return 17
		}
		if len(args) != 4 || args[3] != "-m" {
			return 99
		}
		arch, ok := os.LookupEnv(runCleanupTestArchEnv)
		if !ok {
			arch = "x86_64"
		}
		fmt.Println(arch)
	case "probe-vim", "probe-tmux":
		if fail == command {
			fmt.Fprintln(os.Stderr, runCleanupProbeDiagnostic)
			return 1
		}
		fmt.Println("found")
	case "cp":
		addr, err := os.ReadFile(filepath.Join(dir, "cdr-address"))
		if err != nil {
			return 99
		}
		// Prove that CDR was accepting connections when setup failed.
		conn, err := net.DialTimeout("tcp", string(addr), time.Second)
		if err != nil {
			return 99
		}
		conn.Close()
		return 23
	case "stop":
		if !slices.Equal(args, []string{"stop", "test-container"}) {
			return 99
		}
		time.Sleep(100 * time.Millisecond)
		if err := os.Remove(container); err != nil {
			return 99
		}
	default:
		return 99
	}
	return 0
}

func runCleanupTestCDR(dir string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	// Bound the helper's lifetime even if cleanup regresses.
	timer := time.AfterFunc(12*time.Second, func() { listener.Close() })
	defer timer.Stop()
	if err := os.WriteFile(filepath.Join(dir, "cdr-address"), []byte(listener.Addr().String()), 0600); err != nil {
		return err
	}
	fmt.Printf("{\"pid\":%d,\"port\":%d}\n", os.Getpid(), listener.Addr().(*net.TCPAddr).Port)
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		conn.Close()
	}
}
