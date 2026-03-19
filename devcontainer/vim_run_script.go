package devcontainer

import (
	"html/template"
	"strings"
)

type vimRunScriptParams struct {
	VimFileName string
	SendToTcp   string
	TmuxCommand string
}

func renderVimRunScript(templateSource string, params vimRunScriptParams) (string, error) {
	pattern := "pattern"
	tmpl, err := template.New(pattern).Parse(templateSource)
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
