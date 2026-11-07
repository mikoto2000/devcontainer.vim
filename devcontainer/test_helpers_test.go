package devcontainer

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func requireLinuxDocker(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Docker integration test in short mode")
	}
	output, err := exec.CommandContext(t.Context(), docker.Command, "info", "--format", "{{.OSType}}").Output()
	if err != nil || strings.TrimSpace(string(output)) != "linux" {
		t.Skipf("Linux Docker daemon required: %q, %v", output, err)
	}
}

func createTempAppDirs(t *testing.T) (string, string, string, string) {
	t.Helper()

	baseDir := t.TempDir()
	dirFunc := func() (string, error) {
		return baseDir, nil
	}

	appConfigDir, err := util.CreateConfigDirectory(dirFunc, "devcontainer.vim")
	if err != nil {
		t.Fatalf("failed to create temp config dir: %v", err)
	}

	_, binDir, configDirForDocker, configDirForDevcontainer, err := util.CreateCacheDirectory(dirFunc, "devcontainer.vim")
	if err != nil {
		t.Fatalf("failed to create temp cache dir: %v", err)
	}

	return appConfigDir, binDir, configDirForDocker, configDirForDevcontainer
}

func requireTestBinary(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "test", "resource", "bin", name)
	if !util.IsExists(path) {
		t.Skipf("test binary not found: %s", path)
	}
	return path
}
