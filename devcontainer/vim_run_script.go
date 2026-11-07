package devcontainer

import (
	"strings"
	"text/template"
)

type vimRunScriptParams struct {
	VimFileName string
	SendToTcp   string
	UseTmux     bool
	TmuxCommand string
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
