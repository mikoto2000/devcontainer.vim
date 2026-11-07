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
	if !strings.Contains(script, `'vim' --cmd "let g:devcontainer_vim = v:true" -S '/SendToTcp.vim' -S /vimrc $*`) {
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
	if !strings.Contains(script, `'vim' --cmd "let g:devcontainer_vim = v:true" -S '/SendToTcp.vim' -S /vimrc $*`) {
		t.Fatalf("vim launch command not found in script: %s", script)
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
	clipboard := "SendToTcp" + special + ".vim"
	for _, path := range []string{vim, tmux} {
		if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$0\" \"$@\"\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range []struct {
		name    string
		useTmux bool
	}{
		{name: "direct"},
		{name: "tmux", useTmux: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			script, err := renderVimRunScript(vimRunX8664System, vimRunScriptParams{
				VimFileName: vim,
				SendToTcp:   clipboard,
				UseTmux:     tt.useTmux,
				TmuxCommand: tmux,
			})
			if err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command("/bin/sh", "-c", script).CombinedOutput()
			if err != nil {
				t.Fatalf("launcher failed: %v\n%s", err, out)
			}
			got := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
			want := []string{vim, "--cmd", "let g:devcontainer_vim = v:true", "-S", "/" + clipboard, "-S", "/vimrc"}
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
