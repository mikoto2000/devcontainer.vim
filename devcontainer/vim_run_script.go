package devcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Docker copies the launcher into the container as root, but the editor may
// run as the workspace user.
const vimRunScriptMode = 0755

func writeVimRunScript(path, script string) error {
	dir, err := os.MkdirTemp(filepath.Dir(path), ".vim-run-*")
	if err != nil {
		return err
	}
	defer os.Remove(dir)

	staged := filepath.Join(dir, filepath.Base(path))
	defer os.Remove(staged)
	if err := os.WriteFile(staged, []byte(script), vimRunScriptMode); err != nil {
		return err
	}
	// A rename replaces the file without following a symlink or removing a directory.
	return os.Rename(staged, path)
}

type vimRunScriptParams struct {
	VimFileName string
	SendToTcp   string
	UseTmux     bool
	TmuxCommand string
}

func selectVimRunTemplate(hostOS, containerArch string, useSystemVim bool) string {
	switch {
	case useSystemVim:
		return vimRunX8664System
	case containerArch != "amd64":
		return vimRunAarch64
	case hostOS == "darwin":
		// The installer selects static Vim for amd64 containers on macOS.
		return vimRunX8664Static
	default:
		return vimRunX8664AppImage
	}
}

func renderVimRunScript(templateSource string, params vimRunScriptParams) (string, error) {
	tmpl, err := template.New("vimRun").Funcs(template.FuncMap{"shellQuote": shellQuote}).Parse(templateSource)
	if err != nil {
		return "", err
	}

	var script strings.Builder
	err = tmpl.Execute(&script, params)
	if err != nil {
		return "", err
	}
	return script.String(), nil
}
