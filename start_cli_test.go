package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

const startCLIFixtureEnv = "DEVCONTAINER_VIM_TEST_START_CLI_DIR"

func TestStartCLIDisablesPortForwarding(t *testing.T) {
	dir := isolateCLIHome(t)
	t.Setenv("WSL_INTEROP", "")
	t.Setenv("WSL_DISTRO_NAME", "")
	if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envDevcontainerVimType, "")
	t.Setenv(envDevcontainerShellType, "")
	t.Setenv(startCLIFixtureEnv, dir)
	cache, err := os.UserCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(cache, appName, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(dir, "workspace")
	config := filepath.Join(workspace, ".devcontainer", "devcontainer.json")
	if err := os.MkdirAll(filepath.Dir(config), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte(`{"forwardPorts":[8080]}`), 0600); err != nil {
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
	services := tools.DefaultInstallerUseServices{}
	for name, data := range map[string][]byte{
		dockerName:                            binary,
		tools.DEVCONTAINER(services).FileName: binary,
		tools.CDR(services).FileName:          nil,
		"port-forwarder-container_amd64":      nil,
	} {
		if err := os.WriteFile(filepath.Join(bin, name), data, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestCLIProcess$", "--", "--nopf", "--nocdr", "--notmux", "start", workspace)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
	if output, err := cmd.CombinedOutput(); err != nil {
		unexpected, _ := os.ReadFile(filepath.Join(dir, "unexpected-start-command"))
		t.Fatalf("start with --nopf failed: %v\n%s%s", err, output, unexpected)
	}
	if _, err := os.Stat(filepath.Join(dir, "editor-launched")); err != nil {
		t.Fatalf("start did not reach the editor launch: %v", err)
	}
}

func runStartCLIFixture(dir string, argv []string) error {
	workspace := filepath.Join(dir, "workspace")
	args := argv[1:]
	if strings.TrimSuffix(filepath.Base(argv[0]), ".exe") == "devcontainer" {
		switch {
		case slices.Equal(args, []string{"read-configuration", "--workspace-folder", workspace}):
			return json.NewEncoder(os.Stdout).Encode(map[string]any{
				"configuration": map[string]any{
					"configFilePath": map[string]string{"fsPath": filepath.Join(workspace, ".devcontainer", "devcontainer.json")},
					"forwardPorts":   []int{8080},
				},
			})
		case len(args) == 5 && args[0] == "up" && args[1] == "--override-config" && args[3] == "--workspace-folder" && args[4] == workspace:
			fmt.Println(`{"outcome":"success","containerId":"fixture-container"}`)
			return nil
		case slices.Equal(args, []string{"exec", "--container-id", "fixture-container", "--workspace-folder", workspace, "/VimRun.sh"}):
			return os.WriteFile(filepath.Join(dir, "editor-launched"), nil, 0600)
		}
	} else {
		switch {
		case slices.Equal(args, []string{"exec", "fixture-container", "uname", "-m"}):
			fmt.Println("x86_64")
			return nil
		case len(args) == 7 && slices.Equal(args[:4], []string{"exec", "fixture-container", "sh", "-c"}) && args[5] == "devcontainer-vim-tool-probe" && args[6] == "vim":
			fmt.Println("found")
			return nil
		case len(args) == 3 && args[0] == "cp":
			if args[2] == "fixture-container:/" && (filepath.Base(args[1]) == "SendToTcp.vim" || filepath.Base(args[1]) == "vimrc") ||
				args[2] == "fixture-container:/VimRun.sh" && filepath.Base(args[1]) == "VimRun.sh" {
				return nil
			}
		}
	}
	return fmt.Errorf("unexpected fixture command with --nopf: %q", args)
}
