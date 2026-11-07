package tools

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

const nativeCdrTestEnv = "DEVCONTAINER_VIM_NATIVE_CDR_TEST"
const wslpathTestEnv = "DEVCONTAINER_VIM_TEST_WSLPATH"
const cdrTestPollInterval = 5 * time.Millisecond

func nativeCdrFixture(t *testing.T, mode string) (string, string, string) {
	t.Helper()
	t.Setenv(nativeCdrTestEnv, mode)
	executable := copyTestExecutable(t)
	dir := filepath.Dir(executable)
	return executable, filepath.Join(dir, "pid"), filepath.Join(dir, "port")
}

func TestRunCdrForNativeStarts(t *testing.T) {
	executable, pidFile, portFile := nativeCdrFixture(t, "ready")
	pid, port, err := runCdrForNative(executable, pidFile, portFile, 5*time.Second, cdrTestPollInterval)
	if err != nil {
		t.Fatal(err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Error(err)
		}
		process.Release()
	})
	if pid != readNativeCdrFixtureNumber(t, pidFile) || port != readNativeCdrFixtureNumber(t, portFile) {
		t.Fatalf("unexpected CDR process info: pid=%d port=%d", pid, port)
	}
}

func TestRunCdrTimesOutWhenStartupNeverCompletes(t *testing.T) {
	executable, pidFile, portFile := nativeCdrFixture(t, "timeout")
	pid, port, err := runCdrForNative(executable, pidFile, portFile, 2*time.Second, cdrTestPollInterval)
	if !IsCdrStartupTimeout(err) || pid != 0 || port != 0 {
		t.Fatalf("expected timeout without process info, got pid=%d port=%d err=%v", pid, port, err)
	}
	if !strings.Contains(err.Error(), "waiting for CDR startup") {
		t.Errorf("timeout error lost subprocess diagnostics: %v", err)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(readNativeCdrFixtureNumber(t, portFile)))
	if conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("CDR was still listening after the startup timeout")
	}
}

func TestRunCdrForWsl(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exercises the Linux side of WSL")
	}
	for _, mode := range []string{"ready", "wsl-timeout", "wsl-exit", "conversion-failure"} {
		t.Run(mode, func(t *testing.T) {
			timeout := 5 * time.Second
			if mode == "wsl-timeout" {
				timeout = 500 * time.Millisecond
			}
			executable, _, _ := nativeCdrFixture(t, mode)
			dir := filepath.Dir(executable)
			t.Chdir(dir)
			fakeConverter := filepath.Join(dir, "wslpath")
			binary, err := os.ReadFile(executable)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(fakeConverter, binary, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv(wslpathTestEnv, mode)
			cdr := filepath.Join(dir, "receiver ' & $(touch marker).exe")
			if err := os.Rename(executable, cdr); err != nil {
				t.Fatal(err)
			}
			pidFile := filepath.Join(dir, "pid ' & $(touch marker)")
			portFile := filepath.Join(dir, "port ' & $(touch marker)")
			pid, port, err := runCdrForWsl(cdr, pidFile, portFile, timeout, cdrTestPollInterval)
			if _, err := os.Stat("marker"); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("shell substitution was executed: %v", err)
			}
			switch mode {
			case "ready":
				if err != nil {
					t.Fatal(err)
				}
				process, err := os.FindProcess(pid)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					_ = process.Kill()
					_ = process.Release()
				})
				if pid != readNativeCdrFixtureNumber(t, pidFile) || port != readNativeCdrFixtureNumber(t, portFile) {
					t.Errorf("unexpected CDR process info: pid=%d port=%d", pid, port)
				}
			case "wsl-timeout":
				if !IsCdrStartupTimeout(err) || pid != 0 || port != 0 {
					t.Fatalf("expected a timeout after cleanup, got pid=%d port=%d err=%v", pid, port, err)
				}
				listenPort := readNativeCdrFixtureNumber(t, portFile+".listener")
				address := net.JoinHostPort("127.0.0.1", strconv.Itoa(listenPort))
				if conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond); err == nil {
					conn.Close()
					t.Fatal("CDR was still listening after the startup timeout")
				}
			case "wsl-exit":
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(err.Error(), "fixture exited early") {
					t.Errorf("early startup failure lost its exit status or diagnostics: %v", err)
				}
			case "conversion-failure":
				if err == nil || !strings.Contains(err.Error(), "convert CDR path") {
					t.Errorf("expected a path conversion error, got %v", err)
				}
				if _, err := os.Stat(pidFile); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("CDR started after path conversion failed: %v", err)
				}
			}
		})
	}
}

func readNativeCdrFixtureNumber(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(string(data))
	if err != nil || number <= 0 {
		t.Fatalf("invalid fixture value in %s: %q", path, data)
	}
	return number
}

func TestMain(m *testing.M) {
	if mode := os.Getenv(wslpathTestEnv); mode != "" && filepath.Base(os.Args[0]) == "wslpath" {
		if mode == "conversion-failure" || len(os.Args) != 3 || os.Args[1] != "-w" || !filepath.IsAbs(os.Args[2]) {
			os.Exit(99)
		}
		fmt.Printf("%s\r\n", os.Args[2])
		os.Exit(0)
	}
	if mode := os.Getenv(nativeCdrTestEnv); mode != "" && len(os.Args) == 6 {
		args := os.Args[1:]
		if args[0] == "--random-port" {
			args = append(slices.Clone(args[1:]), args[0])
		}
		if args[0] != "--pid-file" || args[2] != "--port-file" || args[4] != "--random-port" {
			os.Exit(99)
		}
		if err := runNativeCdrFixture(mode, args[1], args[3]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runNativeCdrFixture(mode, pidFile, portFile string) error {
	if mode == "wsl-exit" {
		return errors.New("fixture exited early")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	pid := os.Getpid()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return err
	}
	if mode == "wsl-timeout" {
		portFile += ".listener"
	}
	if err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0600); err != nil {
		return err
	}
	if mode == "ready" {
		fmt.Printf("{\"pid\":%d,", pid)
		time.Sleep(40 * time.Millisecond)
		fmt.Printf("\"port\":%d}\n", port)
	} else {
		fmt.Fprintln(os.Stderr, "waiting for CDR startup")
	}
	// Bound the fixture lifetime even if the caller does not terminate it.
	time.Sleep(15 * time.Second)
	return nil
}

func TestGetProcessInfoRejectsZeroPidOrPort(t *testing.T) {
	output := `{"pid":0,"address":"0.0.0.0","port":0}`
	pid, _, port, err := GetProcessInfo(output)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if pid != 0 || port != 0 {
		t.Fatalf("expected zero pid and port, got pid=%d port=%d", pid, port)
	}
}
