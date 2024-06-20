package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// oras のツール情報
var ORAS Tool = Tool{
	FileName: orasFileName,
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("oras-project", "oras")
		if err != nil {
			panic(err)
		}
		version := latestTagName[1:]

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(downloadURLOrasPattern)
		if err != nil {
			panic(err)
		}

		tmplParams := map[string]string{"TagName": latestTagName, "Version": version}
		var downloadURL strings.Builder
		err = tmpl.Execute(&downloadURL, tmplParams)
		if err != nil {
			panic(err)
		}
		return downloadURL.String()
	},
	installFunc: func(downloadURL string, filePath string) (string, error) {
		return orasInstall(downloadURL, filePath)
	},
}
