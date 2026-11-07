package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const runDockerLogEnv = "DEVCONTAINER_VIM_TEST_DOCKER_LOG"
const configCLIFixtureEnv = "DEVCONTAINER_VIM_TEST_CONFIG_CLI"

func TestMain(m *testing.M) {
	if dir := os.Getenv(startCLIFixtureEnv); dir != "" {
		switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
		case "devcontainer", "docker":
			if err := runStartCLIFixture(dir, os.Args); err != nil {
				_ = os.WriteFile(filepath.Join(dir, "unexpected-start-command"), []byte(err.Error()), 0600)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(99)
			}
			os.Exit(0)
		}
	}
	if os.Getenv(configCLIFixtureEnv) == "1" {
		if len(os.Args) != 4 || os.Args[1] != "read-configuration" || os.Args[2] != "--workspace-folder" {
			fmt.Fprintln(os.Stderr, "unexpected read-configuration arguments:", os.Args[1:])
			os.Exit(99)
		}
		path, err := filepath.Abs(filepath.Join(os.Args[3], ".devcontainer", "devcontainer.json"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(99)
		}
		response := devcontainer.ReadConfigurationCommandResult{
			Configuration: devcontainer.Configuration{ConfigFilePath: devcontainer.ConfigFilePath{FsPath: path}},
		}
		if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(99)
		}
		os.Exit(0)
	}
	if log := os.Getenv(runDockerLogEnv); log != "" && strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "docker" {
		if err := os.WriteFile(log, []byte(strings.Join(os.Args[1:], "\n")), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(99)
	}
	os.Exit(m.Run())
}

func isolateCLIHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	cache := filepath.Join(dir, "cache")
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("APPDATA", config)
	t.Setenv("LOCALAPPDATA", cache)
	return dir
}

func TestRunWithoutImage(t *testing.T) {
	dir := isolateCLIHome(t)
	configBase, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Join(configBase, appName)
	binDir := filepath.Join(cacheBase, appName, "bin")
	for _, path := range []string{configDir, binDir} {
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	// Missing arguments must be reported before runargs is parsed.
	if err := os.WriteFile(filepath.Join(configDir, "runargs"), []byte(";"), 0600); err != nil {
		t.Fatal(err)
	}
	// Keep a regressed CLI from downloading tools before it reaches fake Docker.
	cdr := tools.CDR(tools.DefaultInstallerUseServices{}).FileName
	if err := os.WriteFile(filepath.Join(binDir, cdr), nil, 0700); err != nil {
		t.Fatal(err)
	}

	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	dockerName := "docker"
	if runtime.GOOS == "windows" {
		dockerName += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, dockerName), binary, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	dockerLog := filepath.Join(dir, "docker.log")
	t.Setenv(runDockerLogEnv, dockerLog)

	cmd := exec.Command(self, "-test.run=^TestCLIProcess$", "--", "run")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 ||
		!strings.Contains(string(output), "Usage: devcontainer.vim run <IMAGE_OR_CONTAINER>") {
		t.Errorf("run without image = %v, output %q, want usage and exit code 1", err, output)
	}
	if data, err := os.ReadFile(dockerLog); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Docker was invoked for run without image: arguments %q, read error %v", data, err)
	}
}

func TestStartWithoutWorkspace(t *testing.T) {
	dir := isolateCLIHome(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestCLIProcess$", "--", "start")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 ||
		!strings.Contains(string(output), "Usage: devcontainer.vim start <WORKSPACE_FOLDER>") {
		t.Fatalf("start without workspace = %v, output %q, want usage and exit code 1", err, output)
	}
}

func TestGeneratedConfigDoesNotOverrideEditorPagerOrShell(t *testing.T) {
	dir := isolateCLIHome(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "devcontainer.vim.json")
	cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestCLIProcess$", "--", "config", "-g", "-o", path)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config generation failed: %v\n%s", err, output)
	}
	data, err := util.ParseJwcc(path)
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		RemoteEnv map[string]string `json:"remoteEnv"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if config.RemoteEnv["TERM"] != "xterm-256color" {
		t.Fatalf("missing terminal setting in generated remoteEnv: %v", config.RemoteEnv)
	}
	for _, name := range []string{"EDITOR", "PAGER", "SHELL"} {
		if value, ok := config.RemoteEnv[name]; ok {
			t.Errorf("generated config overrides %s with %q", name, value)
		}
	}
}

func TestConfigDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not applied on Windows")
	}
	dir := isolateCLIHome(t)
	configBase, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(cacheBase, appName)
	outputDir := filepath.Join(dir, "project", ".devcontainer")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, "-test.run=^TestCLIProcess$", "--", "config", "-g", "-o", filepath.Join(outputDir, "devcontainer.vim.json"))
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("config generation failed: %v\n%s", err, output)
	}
	assertMode := func(mode os.FileMode, paths ...string) {
		t.Helper()
		// Compare with a directory created under the same process umask.
		ref := filepath.Join(t.TempDir(), "reference")
		if err := os.Mkdir(ref, mode); err != nil {
			t.Fatal(err)
		}
		want, err := os.Stat(ref)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			got, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Mode().Perm() != want.Mode().Perm() {
				t.Errorf("permissions of %q = %04o, want %04o", path, got.Mode().Perm(), want.Mode().Perm())
			}
		}
	}
	assertMode(0700, filepath.Join(configBase, appName), cache, filepath.Join(cache, "bin"),
		filepath.Join(cache, "config"), filepath.Join(cache, "config", "docker"), filepath.Join(cache, "config", "devcontainer"))
	assertMode(0755, filepath.Dir(outputDir), outputDir)
}

func TestValidateWorkspaceFolder(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")
	for _, tc := range []struct {
		name string
		path string
		want string
	}{
		{"directory", dir, ""},
		{"missing", missing, "workspace folder not found: " + missing},
		{"file", file, "workspace folder is not a directory: " + file},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateWorkspaceFolder(tc.path)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("validateWorkspaceFolder() = %v, want nil", err)
				}
			} else if err == nil || err.Error() != tc.want {
				t.Fatalf("validateWorkspaceFolder() = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestWorkspaceCommandsRejectMissingFolder(t *testing.T) {
	dir := isolateCLIHome(t)
	missing := filepath.Join(dir, "missing workspace")
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, command := range [][]string{{"start"}, {"down"}, {"templates", "apply"}} {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			args := append([]string{"-test.run=^TestCLIProcess$", "--"}, command...)
			cmd := exec.CommandContext(t.Context(), self, append(args, missing)...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
			output, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			want := "Error: workspace folder not found: " + missing
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || !strings.Contains(string(output), want) {
				t.Fatalf("CLI returned %v, output %q, want exit code 1 and %q", err, output, want)
			}
		})
	}
}

func TestToolDownloadRejectsUnknownArchitecture(t *testing.T) {
	dir := isolateCLIHome(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, "-test.run=^TestCLIProcess$", "--", "tool", "vim", "download", "--arch", "riscv64")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	output, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 ||
		!strings.Contains(string(output), `Error installing vim: Unknown Architecture: "riscv64"`) {
		t.Fatalf("tool download with unknown architecture = %v, output %q, want architecture error and exit code 1", err, output)
	}
}

func TestCLIProcess(t *testing.T) {
	if os.Getenv("DEVCONTAINER_VIM_TEST_CLI") != "1" {
		return
	}
	delimiter := slices.Index(os.Args, "--")
	if delimiter < 0 {
		t.Fatal("missing CLI argument separator")
	}
	os.Args = append([]string{os.Args[0]}, os.Args[delimiter+1:]...)
	main()
}

func TestCreateConfigFile(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv(configCLIFixtureEnv, "1")
	devcontainerPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	workspaceFolder := filepath.Join("project", "workspace with spaces")
	workspaceAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(workspaceFolder, ".devcontainer", "devcontainer.json")
	if err := os.MkdirAll(filepath.Dir(source), 0700); err != nil {
		t.Fatal(err)
	}
	const config = "{\"name\":\"test\"}\n"
	if err := os.WriteFile(source, []byte(config), 0600); err != nil {
		t.Fatal(err)
	}

	configDirForDevcontainer := filepath.Join(root, "cache")
	configFilePath, err := devcontainer.CreateConfigFile(devcontainerPath, workspaceFolder, configDirForDevcontainer)
	if err != nil {
		t.Fatal(err)
	}
	hash := md5.Sum([]byte(workspaceAbs))
	want := filepath.Join(configDirForDevcontainer, fmt.Sprintf("%x", hash), "devcontainer.json")
	if want != configFilePath {
		t.Fatalf("config path = %q, want %q", configFilePath, want)
	}
	if data, err := os.ReadFile(configFilePath); err != nil || string(data) != config {
		t.Fatalf("generated config = %q, %v, want %q", data, err, config)
	}
}
