package tools

import (
	"errors"
	"runtime"
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// Vim のダウンロード URL
const vimAppImageDownloadURLPattern = "https://github.com/vim/vim-appimage/releases/download/{{ .TagName }}/Vim-{{ .TagName }}.glibc2.29-x86_64.AppImage"
const vimX64StaticDownloadURLPattern = "https://github.com/mikoto2000/vim-static/releases/download/{{ .TagName }}/vim-{{ .TagName }}-x86_64.tar.gz"
const vimArmStaticDownloadURLPattern = "https://github.com/mikoto2000/vim-static/releases/download/{{ .TagName }}/vim-{{ .TagName }}-aarch64.tar.gz"

// Vim のツール情報
var VIM Tool = Tool{
	FileName: "vim",
	CalculateDownloadURL: func(containerArch string) (string, error) {
		if containerArch == "amd64" || containerArch == "x86_64" {
			if runtime.GOOS != "darwin" {
				latestTagName, err := util.GetLatestReleaseFromGitHub("vim", "vim-appimage")
				if err != nil {
					return "", err
				}

				pattern := "pattern"
				tmpl, err := template.New(pattern).Parse(vimAppImageDownloadURLPattern)
				if err != nil {
					return "", err
				}

				tmplParams := map[string]string{"TagName": latestTagName}
				var downloadURL strings.Builder
				err = tmpl.Execute(&downloadURL, tmplParams)
				if err != nil {
					return "", err
				}
				return downloadURL.String(), nil
			} else {
				latestTagName, err := util.GetLatestReleaseFromGitHub("vim", "vim-appimage")
				if err != nil {
					return "", err
				}

				pattern := "pattern"
				tmpl, err := template.New(pattern).Parse(vimX64StaticDownloadURLPattern)
				if err != nil {
					return "", err
				}

				tmplParams := map[string]string{"TagName": latestTagName}
				var downloadURL strings.Builder
				err = tmpl.Execute(&downloadURL, tmplParams)
				if err != nil {
					return "", err
				}
				return downloadURL.String(), nil
			}
		} else if containerArch == "arm64" || containerArch == "aarch64" {
			latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "vim-static")
			if err != nil {
				return "", err
			}

			pattern := "pattern"
			tmpl, err := template.New(pattern).Parse(vimArmStaticDownloadURLPattern)
			if err != nil {
				return "", err
			}

			tmplParams := map[string]string{"TagName": latestTagName}
			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				return "", err
			}
			return downloadURL.String(), nil
		} else {
			return "", errors.New("Unknown Architecture")
		}
	},
	installFunc: func(downloadURL string, filePath string, containerArch string) (string, error) {
		return SimpleInstall(downloadURL, filePath)
	},
}
