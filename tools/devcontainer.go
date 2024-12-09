package tools

import (
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

			tmplParams := map[string]string{"TagName": latestTagName}
			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				return "", err
			}
			return downloadURL.String(), nil
		},
		installFunc: func(downloadURL string, filePath string, _ string) (string, error) {
			return services.SimpleInstall(downloadURL, filePath)
		},
	}
}
