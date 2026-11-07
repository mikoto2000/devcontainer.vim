package devcontainer

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const stopTestDirEnv = "DEVCONTAINER_VIM_STOP_TEST_DIR"

type stopTestFixture struct {
	Workspace string
	Config    Configuration
	Unlabeled bool
	Fail      bool
}

type stopTestCall struct {
	Dir  string
	Args []string
}

func TestStopAndDownWaitForDocker(t *testing.T) {
	for _, mode := range []string{"string", "array", "unlabeled array", "docker"} {
		for _, action := range []string{"stop", "down"} {
			for _, fail := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/fail=%t", mode, action, fail), func(t *testing.T) {
					dir := t.TempDir()
					t.Chdir(dir)
					workspace := filepath.Join(dir, "workspace")
					configFile := filepath.Join(workspace, ".devcontainer", "devcontainer.json")
					composeDir := filepath.Dir(configFile)
					base := filepath.Join(composeDir, "app compose.yaml")
					override := filepath.Join(workspace, "override.yaml")
					if err := os.MkdirAll(composeDir, 0755); err != nil {
						t.Fatal(err)
					}

					fixture := stopTestFixture{
						Workspace: workspace, Fail: fail, Unlabeled: mode == "unlabeled array",
						Config: Configuration{ConfigFilePath: ConfigFilePath{FsPath: configFile}},
					}
					files := []string{base}
					switch mode {
					case "string":
						fixture.Config.DockerComposeFile = filepath.Base(base)
					case "array", "unlabeled array":
						fixture.Config.DockerComposeFile = []string{filepath.Base(base), override}
						files = append(files, override)
					}

					devcontainerPath := installStopTestCommands(t, dir, fixture)

					cache := filepath.Join(dir, "cache")
					configDir, err := util.GetConfigDir(cache, "workspace")
					if err != nil {
						t.Fatal(err)
					}
					if err := os.MkdirAll(configDir, 0755); err != nil {
						t.Fatal(err)
					}
					args := []string{"--nocdr", "workspace"}
					if action == "stop" {
						err = Stop(args, devcontainerPath, cache)
					} else {
						err = Down(args, devcontainerPath, cache)
					}
					if fail {
						var exit *exec.ExitError
						if !errors.As(err, &exit) || exit.ExitCode() != 23 || !strings.Contains(err.Error(), "simulated Docker failure") {
							t.Errorf("error = %v, want Docker exit code 23 with diagnostics", err)
						}
					} else if err != nil {
						t.Error(err)
					}
					if _, err := os.Stat(filepath.Join(dir, "finished")); err != nil {
						t.Errorf("Docker had not finished when %s returned: %v", action, err)
					}
					_, statErr := os.Stat(configDir)
					if action == "down" && !fail {
						if !errors.Is(statErr, os.ErrNotExist) {
							t.Errorf("cache after down: %v, want removed", statErr)
						}
					} else if statErr != nil {
						t.Errorf("cache should be retained: %v", statErr)
					}
					cwd, err := os.Getwd()
					if err != nil || cwd != dir {
						t.Errorf("cwd = %q, %v, want %q", cwd, err, dir)
					}

					var want stopTestCall
					if mode == "docker" {
						want = stopTestCall{Dir: dir, Args: []string{"stop", "container-id"}}
						if action == "down" {
							want.Args = []string{"rm", "-f", "container-id"}
						}
					} else {
						flags := []string{"compose"}
						for _, file := range files {
							flags = append(flags, "-f", file)
						}
						want = stopTestCall{Dir: workspace, Args: append(slices.Clone(flags), "-p", "workspace_devcontainer", action)}
						if fixture.Unlabeled {
							assertStopTestCall(t, filepath.Join(dir, "compose-ps"), stopTestCall{
								Dir: workspace, Args: append(flags, "ps", "--all", "--format", "json"),
							})
						}
					}
					assertStopTestCall(t, filepath.Join(dir, "finished"), want)
				})
			}
		}
	}
}

func TestDownCdrPidFile(t *testing.T) {
	for _, pid := range []string{"0", "-1", "-2", "positive"} {
		t.Run(pid, func(t *testing.T) {
			dir := t.TempDir()
			workspace := filepath.Join(dir, "workspace")
			if err := os.Mkdir(workspace, 0700); err != nil {
				t.Fatal(err)
			}
			cache := filepath.Join(dir, "cache")
			configDir, err := util.GetConfigDir(cache, workspace)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(configDir, 0700); err != nil {
				t.Fatal(err)
			}
			fixture := stopTestFixture{Workspace: workspace}
			pidFile := filepath.Join(configDir, "pid")
			if pid != "positive" {
				// Route any attempted signal to a recorder so a regression is safe to test.
				devcontainerPath := installStopTestCommands(t, dir, fixture, "powershell.exe")
				t.Setenv("WSL_DISTRO_NAME", "cdr-test")
				if err := os.WriteFile(pidFile, []byte(pid), 0600); err != nil {
					t.Fatal(err)
				}
				err := Down([]string{workspace}, devcontainerPath, cache)
				if err == nil || !strings.Contains(err.Error(), "invalid clipboard-data-receiver PID "+pid+": must be positive") {
					t.Errorf("Down error = %v, want invalid PID %s", err, pid)
				}
				if _, err := os.Stat(filepath.Join(dir, "cdr-signal-attempt")); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("Down attempted to signal invalid PID %s: stat error = %v", pid, err)
				}
				if data, err := os.ReadFile(pidFile); err != nil || string(data) != pid {
					t.Errorf("PID file after failed Down = %q, %v, want %q", data, err, pid)
				}
				return
			}

			devcontainerPath := installStopTestCommands(t, dir, fixture, "run-test-cdr")
			// Exercise the native signal path even when running under WSL.
			t.Setenv("WSL_DISTRO_NAME", "")
			if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
				t.Fatal(err)
			}
			cdr := exec.Command("run-test-cdr")
			cdr.Env = append(os.Environ(), runCleanupTestDirEnv+"="+dir)
			if err := cdr.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan struct{})
			var waitErr error
			go func() {
				waitErr = cdr.Wait()
				close(done)
			}()
			t.Cleanup(func() {
				_ = cdr.Process.Kill()
				<-done
			})
			if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cdr.Process.Pid)), 0600); err != nil {
				t.Fatal(err)
			}
			if err := Down([]string{workspace}, devcontainerPath, cache); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
				var exit *exec.ExitError
				if !errors.As(waitErr, &exit) || exit.ExitCode() == 99 {
					t.Errorf("CDR exit = %v, want termination by Down", waitErr)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("CDR is still running after Down")
			}
			if _, err := os.Stat(configDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("cache after Down: %v, want removed", err)
			}
		})
	}
}

func installStopTestCommands(t *testing.T, dir string, fixture stopTestFixture, extraNames ...string) string {
	t.Helper()
	binDir := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	commandPath := func(name string) string {
		if runtime.GOOS == "windows" && !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
		return filepath.Join(binDir, name)
	}
	for _, name := range append([]string{"docker", "test-devcontainer"}, extraNames...) {
		if err := os.WriteFile(commandPath(name), binary, 0700); err != nil {
			t.Fatal(err)
		}
	}
	data, err := json.Marshal(fixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(stopTestDirEnv, dir)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return commandPath("test-devcontainer")
}

func assertStopTestCall(t *testing.T, path string, want stopTestCall) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got stopTestCall
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Docker call = %+v, want %+v", got, want)
	}
}

func runStopTestCommand(dir string, argv []string) int {
	data, err := os.ReadFile(filepath.Join(dir, "fixture.json"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 99
	}
	var fixture stopTestFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 99
	}
	switch strings.TrimSuffix(filepath.Base(argv[0]), ".exe") {
	case "powershell":
		if err := os.WriteFile(filepath.Join(dir, "cdr-signal-attempt"), []byte(strings.Join(argv[1:], " ")), 0600); err != nil {
			return 99
		}
		return 0
	case "test-devcontainer":
		json.NewEncoder(os.Stdout).Encode(ReadConfigurationCommandResult{Configuration: fixture.Config})
		return 0
	}
	args := argv[1:]
	switch args[0] {
	case "ps":
		if !slices.Contains(args, "--all") {
			fmt.Println(`{"ID":"container-id"}`)
		} else if !fixture.Unlabeled {
			encoder := json.NewEncoder(os.Stdout)
			encoder.Encode(map[string]string{"workspace": fixture.Workspace, "config": fixture.Config.ConfigFilePath.FsPath + ".other", "project": "other-project"})
			encoder.Encode(map[string]string{"workspace": fixture.Workspace, "config": fixture.Config.ConfigFilePath.FsPath, "project": "workspace_devcontainer"})
		}
		return 0
	case "stop", "rm", "compose":
	default:
		return 99
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 99
	}
	data, err = json.Marshal(stopTestCall{Dir: cwd, Args: args})
	if err != nil {
		return 99
	}
	if args[0] == "compose" && len(args) >= 5 && args[len(args)-4] == "ps" {
		if err := os.WriteFile(filepath.Join(dir, "compose-ps"), data, 0600); err != nil {
			return 99
		}
		fmt.Println(`[{"Project":"workspace_devcontainer"}]`)
		return 0
	}
	time.Sleep(75 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(dir, "finished"), data, 0600); err != nil {
		return 99
	}
	if fixture.Fail {
		fmt.Fprintln(os.Stderr, "simulated Docker failure")
		return 23
	}
	return 0
}
