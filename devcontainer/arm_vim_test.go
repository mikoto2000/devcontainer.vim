package devcontainer

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const armVimTestDirEnv = "DEVCONTAINER_VIM_ARM_VIM_TEST_DIR"

func TestArmContainerEditor(t *testing.T) {
	// Keep the CDR fixture and the fake CLI on the native exec path under WSL.
	t.Setenv("WSL_DISTRO_NAME", "")
	if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name       string
		run        bool
		systemNvim bool
	}{
		{name: "run/fallback", run: true},
		{name: "start/fallback"},
		{name: "start/system Neovim", systemNvim: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			containerDir := filepath.Join(dir, "container")
			if err := os.Mkdir(containerDir, 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv(armVimTestDirEnv, containerDir)
			t.Setenv("DEVCONTAINER_VIM_ARM_TEST_SYSTEM_NVIM", fmt.Sprint(tc.systemNvim))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv(runCleanupTestDirEnv, dir)

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
			for _, name := range []string{"docker", "test-arm-devcontainer", "run-test-cdr"} {
				if err := os.WriteFile(filepath.Join(dir, name+suffix), binary, 0700); err != nil {
					t.Fatal(err)
				}
			}
			const archive = "static Vim archive"
			for name, content := range map[string]string{
				"vim_aarch64":                      archive,
				"nvim_aarch64":                     "legacy archive under the Neovim cache key",
				"port-forwarder-container_aarch64": "",
				"vimrc":                            "",
			} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}

			vimrc := filepath.Join(dir, "vimrc")
			if tc.run {
				cdr := filepath.Join(dir, "run-test-cdr"+suffix)
				err = Run([]string{"arm-image"}, false, true, true, cdr, dir, true, "", dir, vimrc, nil)
			} else {
				cli := filepath.Join(dir, "test-arm-devcontainer"+suffix)
				config := filepath.Join(dir, "devcontainer.json")
				err = Start(DefaultDevcontainerStartUseService{}, []string{dir}, cli, true, true, true, "", dir, true, "", config, vimrc)
			}
			if err != nil {
				t.Fatal(err)
			}

			read := func(dir, name string) string {
				t.Helper()
				data, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			script := read(containerDir, "launched")
			if tc.systemNvim {
				if !strings.Contains(script, "\n'nvim' --cmd ") || !strings.Contains(script, " -S '/SendToTcp.lua' ") {
					t.Errorf("system Neovim launch script = %q", script)
				}
				if clip := read(containerDir, "SendToTcp.lua"); !strings.Contains(clip, "function SendToCdr(register)") {
					t.Errorf("Neovim clipboard script = %q", clip)
				}
				return
			}

			for _, part := range []string{"tar zxf './vim' -C", "~/vim-static/AppRun --cmd ", " -S '/SendToTcp.vim' "} {
				if !strings.Contains(script, part) {
					t.Errorf("Vim launch script missing %q: %s", part, script)
				}
			}
			if got := read(containerDir, "vim"); got != archive {
				t.Errorf("archive copied to /vim = %q, want %q", got, archive)
			}
			clipboard := read(containerDir, "SendToTcp.vim")
			if !strings.Contains(clipboard, "function! SendToCdr(register) abort") {
				t.Errorf("Vim clipboard script = %q", clipboard)
			}
			if tc.run {
				_, port, err := net.SplitHostPort(read(dir, "cdr-address"))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(clipboard, `ch_open("host.docker.internal:`+port+`"`) {
					t.Errorf("Vim clipboard script does not connect to CDR port %s: %s", port, clipboard)
				}
			}
		})
	}
}

func runArmVimTestCommand(dir string, args []string) error {
	switch args[0] {
	case "run":
		fmt.Println("arm-container")
		return nil
	case "up":
		fmt.Println(`{"outcome":"success","containerId":"arm-container"}`)
		return nil
	case "stop":
		return nil
	case "cp":
		if dest, ok := strings.CutPrefix(args[2], "arm-container:/"); ok {
			// These launch cases disable port forwarding. A cached binary keeps
			// the old install path offline so an attempted copy fails the test.
			if dest == "port-forwarder" {
				return fmt.Errorf("unexpected port-forwarder copy with forwarding disabled")
			}
			if dest == "" {
				dest = filepath.Base(args[1])
			}
			data, err := os.ReadFile(args[1])
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, dest), data, 0600)
		}
	case "exec":
		switch {
		case slices.Equal(args[1:], []string{"-t", "arm-container", "uname", "-m"}):
			fmt.Println("aarch64")
			return nil
		case slices.Equal(args[1:], []string{"-t", "arm-container", "which", "nvim"}):
			if os.Getenv("DEVCONTAINER_VIM_ARM_TEST_SYSTEM_NVIM") == "true" {
				fmt.Println("/usr/bin/nvim")
				return nil
			}
		case args[1] == "--user":
			return nil
		case args[len(args)-1] == "/VimRun.sh":
			data, err := os.ReadFile(filepath.Join(dir, "VimRun.sh"))
			if err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(dir, "launched"), data, 0600)
		}
	}
	return fmt.Errorf("unexpected or unsuccessful fixture command: %v", args)
}
