//go:build windows && amd64

package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// NeoVim のダウンロード URL
const nvimDownloadURLPattern = "https://github.com/neovim/neovim/releases/download/{{ .TagName }}/nvim.appimage"

// NeoVim のツール情報
var NVIM Tool = Tool{
	FileName: "nvim",
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("neovim", "neovim")
		if err != nil {
			panic(err)
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(vimDownloadURLPattern)
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
