//go:build darwin

package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// Vim のダウンロード URL
const vimDownloadURLPattern = "https://github.com/mikoto2000/vim-static/releases/download/{{ .TagName }}/vim-{{ .TagName }}-aarch64.tar.gz"

// Vim のツール情報
var VIM Tool = Tool{
	FileName: "vim",
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "vim-static")
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
		_, err := simpleInstall(downloadURL, filePath)
		if err != nil {
			return "", err
		}

		// TODO, tar.gz を展開
		//err = util.ExtractTarGz(filePath + ".tar.gz" ,filePath)
		//if err != nil {
		//	return "", err
		//}
		return filePath, nil

	},
}

