package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// devcontainer/cli のツール情報
var DEVCONTAINER Tool = Tool{
	FileName: devcontainerFileName,
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "devcontainers-cli")
		if err != nil {
			panic(err)
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(downloadURLDevcontainersCliPattern)
		if err != nil {
			panic(err)
		}

		tmplParams := map[string]string{"TagName": latestTagName}
		var downloadURL strings.Builder
		err = tmpl.Execute(&downloadURL, tmplParams)
		if err != nil {
			panic(err)
		}
		return downloadURL.String()
	},
	installFunc: func(downloadURL string, filePath string) (string, error) {
		return simpleInstall(downloadURL, filePath)
	},
}
