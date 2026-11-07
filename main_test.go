package main

import (
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

func TestMain(m *testing.M) {
	if log := os.Getenv(runDockerLogEnv); log != "" && strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") == "docker" {
		if err := os.WriteFile(log, []byte(strings.Join(os.Args[1:], "\n")), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(99)
	}
	os.Exit(m.Run())
}

func TestRunWithoutImage(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config")
	cache := filepath.Join(dir, "cache")
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("XDG_CONFIG_HOME", config)
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("APPDATA", config)
	t.Setenv("LOCALAPPDATA", cache)

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
	_, binDir, _, configDirForDevcontainer, err := util.CreateCacheDirectory(func() (string, error) {
		return "./test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, _, err := tools.InstallStartTools(tools.DefaultInstallerUseServices{}, binDir)
	if err != nil {
		t.Fatalf("Error installing start tools: %v", err)
	}

	workspaceFolder := "./test/project/TestCreateConfigFile"
	configFilePath, err := devcontainer.CreateConfigFile(devcontainerPath, workspaceFolder, configDirForDevcontainer)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			t.Skipf("configuration file not found: %v", err)
		}
		if strings.Contains(err.Error(), "出力パースに失敗") || strings.Contains(err.Error(), "read-configuration` に失敗") {
			t.Skipf("devcontainer CLI environment issue: %v", err)
		}
		t.Fatalf("error: %v", err)
	}
	want := "test/resource/config/devcontainer/6d0900e89898e089cf294371661aea37/devcontainer.json"
	if want != configFilePath {
		t.Fatalf("error: want %s, but got %s", want, configFilePath)
	}
}
