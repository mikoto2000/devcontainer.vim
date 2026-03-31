package tools

import (
	"runtime"
	"strings"
	"text/template"
)

// devcontainer/cli のツール情報
var DEVCONTAINER = func(services InstallerUseServices) Tool {

	return Tool{
		FileName: devcontainerFileName,
		CalculateDownloadURL: func(_ string) (string, error) {
			latestTagName, err := services.GetLatestReleaseFromGitHub("mikoto2000", "devcontainers-cli")
			if err != nil {
				return "", err
			}

			pattern := "pattern"
			tmpl, err := template.New(pattern).Parse(downloadURLDevcontainersCliPattern)
			if err != nil {
				return "", err
			}

			// GOARCH が amd64 だった場合、 x64 に変換する
			arch := runtime.GOARCH
			if arch == "amd64" {
				arch = "x64"
			}

			tmplParams := map[string]string{
				"TagName": latestTagName,
				"Arch":    arch,
			}
			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				return "", err
			}
			return downloadURL.String(), nil
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: services.Download,
	}
}
