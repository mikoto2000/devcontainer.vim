//go:build linux && amd64

package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// Vim のダウンロード URL
const vimDownloadURLPattern = "https://github.com/vim/vim-appimage/releases/download/{{ .TagName }}/Vim-{{ .TagName }}.glibc2.29-x86_64.AppImage"

// Vim のツール情報
var VIM Tool = Tool{
	FileName: "vim",
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("vim", "vim-appimage")
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


