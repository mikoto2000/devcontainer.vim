package devcontainer

import (
	"strings"
	"testing"
)

func TestRenderVimRunScriptWrapsWithTmux(t *testing.T) {
	script, err := renderVimRunScript(vimRunX8664System, vimRunScriptParams{
		VimFileName: "vim",
		SendToTcp:   "SendToTcp.vim",
		TmuxCommand: "/tmux",
	})
	if err != nil {
		t.Fatalf("renderVimRunScript error: %v", err)
	}

	if !strings.Contains(script, `/tmux -u set-option -g status off \; set-option -g set-clipboard on \; new-session`) {
		t.Fatalf("tmux launch command not found in script: %s", script)
	}
	if !strings.Contains(script, `vim --cmd "let g:devcontainer_vim = v:true" -S /SendToTcp.vim -S /vimrc $*`) {
		t.Fatalf("vim launch command not found in script: %s", script)
	}
}
