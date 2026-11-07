package devcontainer

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestVimLaunchPreservesExistingDirectory(t *testing.T) {
	for _, tt := range []struct {
		name   string
		launch func(dir string) ([]string, error)
	}{
		{"run", func(dir string) ([]string, error) {
			return dockerRunVimArgs("container-id", "vim", "", "SendToTcp.vim", "amd64", true, false, true, "", dir)
		}},
		{"start", func(dir string) ([]string, error) {
			return devcontainerStartVimArgs("container-id", "workspace", "vim", "", "SendToTcp.vim", "amd64", true, false, true, "", dir)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, dir := installForwardTestCommands(t, "[]")
			collision := filepath.Join(dir, "VimRun.sh")
			if err := os.Mkdir(collision, 0700); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(collision, "keep")
			if err := os.WriteFile(marker, []byte("existing contents"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := tt.launch(dir); err == nil {
				t.Error("launch script generation accepted a directory")
			}
			if data, err := os.ReadFile(marker); err != nil || string(data) != "existing contents" {
				t.Errorf("existing directory contents = %q, %v", data, err)
			}
		})
	}
}

func TestWriteVimRunScriptReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "VimRun.sh")
	if err := os.WriteFile(path, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeVimRunScript(path, "new"); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "new" {
		t.Fatalf("replacement contents = %q, %v", data, err)
	}
}

func TestSelectVimRunTemplate(t *testing.T) {
	for _, tc := range []struct {
		name, hostOS, arch, want string
		useSystem                bool
	}{
		{name: "system on ARM", hostOS: "linux", arch: "aarch64", useSystem: true, want: vimRunX8664System},
		{name: "system on macOS", hostOS: "darwin", arch: "amd64", useSystem: true, want: vimRunX8664System},
		{name: "Linux AppImage", hostOS: "linux", arch: "amd64", want: vimRunX8664AppImage},
		{name: "Windows AppImage", hostOS: "windows", arch: "amd64", want: vimRunX8664AppImage},
		{name: "macOS static x86_64", hostOS: "darwin", arch: "amd64", want: vimRunX8664Static},
		{name: "macOS static ARM", hostOS: "darwin", arch: "aarch64", want: vimRunAarch64},
		{name: "Linux static ARM", hostOS: "linux", arch: "aarch64", want: vimRunAarch64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := selectVimRunTemplate(tc.hostOS, tc.arch, tc.useSystem); got != tc.want {
				t.Errorf("unexpected template for host %q, architecture %q, system=%t:\n%s", tc.hostOS, tc.arch, tc.useSystem, got)
			}
		})
	}
}

func TestRenderVimRunScriptWrapsWithTmux(t *testing.T) {
	script, err := renderVimRunScript(vimRunX8664System, vimRunScriptParams{
		VimFileName: "vim",
		SendToTcp:   "SendToTcp.vim",
		UseTmux:     true,
		TmuxCommand: "/tmux",
	})
	if err != nil {
		t.Fatalf("renderVimRunScript error: %v", err)
	}

	if !strings.Contains(script, `'/tmux' -u set-option -g prefix None \; unbind-key C-b \; set-option -g status off \; set-option -g set-clipboard on \; new-session`) {
		t.Fatalf("tmux launch command not found in script: %s", script)
	}
	if !strings.Contains(script, `'vim' --cmd "let g:devcontainer_vim = v:true" -S '/SendToTcp.vim' -S /vimrc "$@"`) {
		t.Fatalf("vim launch command not found in script: %s", script)
	}
}

func TestRenderVimRunScriptWithoutTmux(t *testing.T) {
	script, err := renderVimRunScript(vimRunX8664System, vimRunScriptParams{
		VimFileName: "vim",
		SendToTcp:   "SendToTcp.vim",
		UseTmux:     false,
	})
	if err != nil {
		t.Fatalf("renderVimRunScript error: %v", err)
	}

	if strings.Contains(script, `new-session`) {
		t.Fatalf("tmux launch command should not be found in script: %s", script)
	}
	if !strings.Contains(script, `'vim' --cmd "let g:devcontainer_vim = v:true" -S '/SendToTcp.vim' -S /vimrc "$@"`) {
		t.Fatalf("vim launch command not found in script: %s", script)
	}
}

func TestRenderBundledVimRunScripts(t *testing.T) {
	for _, variant := range []struct {
		name, template, unpack, prepare, editor string
	}{
		{
			name: "AppImage", template: vimRunX8664AppImage,
			unpack: "'/vim archive' --appimage-extract", prepare: "cd -",
			editor: "~/squashfs-root/AppRun",
		},
		{
			name: "x86_64 static", template: vimRunX8664Static,
			unpack: "tar zxf './vim archive' -C ~/", prepare: "mv $(ls -d ~/vim-*-x86_64) ~/vim-static",
			editor: "~/vim-static/AppRun",
		},
		{
			name: "ARM static", template: vimRunAarch64,
			unpack: "tar zxf './vim archive' -C ~/", prepare: "mv $(ls -d ~/vim-*-aarch64) ~/vim-static",
			editor: "~/vim-static/AppRun",
		},
	} {
		for _, mode := range []struct {
			name    string
			useTmux bool
		}{
			{name: "direct"},
			{name: "tmux", useTmux: true},
		} {
			t.Run(variant.name+"/"+mode.name, func(t *testing.T) {
				script, err := renderVimRunScript(variant.template, vimRunScriptParams{
					VimFileName: "vim archive",
					SendToTcp:   "Send To Tcp.vim",
					UseTmux:     mode.useTmux,
					TmuxCommand: "/tmux custom",
				})
				if err != nil {
					t.Fatal(err)
				}
				editor := variant.editor + ` --cmd "let g:devcontainer_vim = v:true" -S '/Send To Tcp.vim' -S /vimrc "$@"`
				launch := "\n" + editor
				if mode.useTmux {
					launch = `new-session -s "devcontainer.vim" -A ` + editor
				}
				for _, want := range []string{variant.unpack, variant.prepare, launch} {
					if !strings.Contains(script, want) {
						t.Errorf("launcher missing %q:\n%s", want, script)
					}
				}
				if hasTmux := strings.Contains(script, "'/tmux custom' -u "); hasTmux != mode.useTmux {
					t.Errorf("tmux command presence = %t, want %t:\n%s", hasTmux, mode.useTmux, script)
				}
				if runtime.GOOS != "windows" {
					cmd := exec.Command("/bin/sh", "-n")
					cmd.Stdin = strings.NewReader(script)
					if output, err := cmd.CombinedOutput(); err != nil {
						t.Errorf("invalid shell script: %v\n%s", err, output)
					}
				}
			})
		}
	}
}

func TestVimRunScriptPreservesShellArguments(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX executable paths and /bin/sh")
	}
	dir := t.TempDir()
	special := ` & <tag> 'quote' $(printf expanded)`
	vim := filepath.Join(dir, "vim"+special)
	tmux := filepath.Join(dir, "tmux"+special)
	staticVim := filepath.Join(dir, "vim-static", "AppRun")
	if err := os.Mkdir(filepath.Dir(staticVim), 0700); err != nil {
		t.Fatal(err)
	}
	clipboard := "SendToTcp" + special + ".vim"
	for _, path := range []string{vim, tmux, staticVim} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$0\" \"$@\"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "glob-match.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	extra := []string{"file with spaces", "", "*.txt"}
	for _, tt := range []struct {
		name, template, editor string
		useTmux                bool
	}{
		{name: "system/direct", template: vimRunX8664System, editor: vim},
		{name: "system/tmux", template: vimRunX8664System, editor: vim, useTmux: true},
		{name: "static/direct", template: vimRunX8664Static, editor: staticVim},
		{name: "static/tmux", template: vimRunX8664Static, editor: staticVim, useTmux: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script, err := renderVimRunScript(tt.template, vimRunScriptParams{
				VimFileName: vim,
				SendToTcp:   clipboard,
				UseTmux:     tt.useTmux,
				TmuxCommand: tmux,
			})
			if err != nil {
				t.Fatal(err)
			}
			if tt.template == vimRunX8664Static {
				// Exercise argv forwarding independently of archive extraction.
				script = strings.TrimSpace(script)
				script = script[strings.LastIndexByte(script, '\n')+1:]
			}
			cmd := exec.Command("/bin/sh", append([]string{"-c", script, "VimRun.sh"}, extra...)...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "HOME="+dir)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("launcher failed: %v\n%s", err, out)
			}
			got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
			want := []string{tt.editor, "--cmd", "let g:devcontainer_vim = v:true", "-S", "/" + clipboard, "-S", "/vimrc"}
			want = append(want, extra...)
			if tt.useTmux {
				want = append([]string{
					tmux, "-u", "set-option", "-g", "prefix", "None", ";",
					"unbind-key", "C-b", ";", "set-option", "-g", "status", "off", ";",
					"set-option", "-g", "set-clipboard", "on", ";",
					"new-session", "-s", "devcontainer.vim", "-A",
				}, want...)
			}
			if !slices.Equal(got, want) {
				t.Errorf("launcher arguments = %q, want %q", got, want)
			}
		})
	}
}
