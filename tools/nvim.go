package tools

import (
	"errors"
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// NeoVim のダウンロード URL
const nvimAppImageDownloadURLPattern = "https://github.com/neovim/neovim/releases/download/{{ .TagName }}/nvim-linux-x86_64.appimage"
const nvimArmStaticDownloadURLPattern = "https://github.com/mikoto2000/vim-static/releases/download/{{ .TagName }}/vim-{{ .TagName }}-aarch64.tar.gz"

// NeoVim のツール情報
var NVIM = func(service InstallerUseServices) Tool {
	return Tool{
		FileName: "nvim",
		CalculateDownloadURL: func(containerArch string) (string, error) {
			if containerArch == "amd64" || containerArch == "x86_64" {
				latestTagName, err := util.GetLatestReleaseFromGitHub("neovim", "neovim")
				if err != nil {
					return "", err
				}

				pattern := "pattern"
				tmpl, err := template.New(pattern).Parse(nvimAppImageDownloadURLPattern)
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
			} else if containerArch == "arm64" || containerArch == "aarch64" {
				latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "vim-static")
				if err != nil {
					return "", err
				}

				pattern := "pattern"
				tmpl, err := template.New(pattern).Parse(nvimArmStaticDownloadURLPattern)
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
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: download,
	}
}
