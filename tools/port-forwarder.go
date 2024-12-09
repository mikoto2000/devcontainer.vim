package tools

import (
	"errors"
	"strings"
	"text/template"
)

const downloadURLPortForwarderContainerAmd64Pattern = "https://github.com/mikoto2000/port-forwarder/releases/download/{{ .TagName }}/port-forwarder-linux-amd64"
const downloadURLPortForwarderContainerArm64Pattern = "https://github.com/mikoto2000/port-forwarder/releases/download/{{ .TagName }}/port-forwarder-linux-arm64"

// コンテナ上で起動する port-forwarder のツール情報
var PortForwarderContainer = func(services InstallerUseServices) Tool {

	return Tool{
		FileName: "port-forwarder-container",
		CalculateDownloadURL: func(containerArch string) (string, error) {
			latestTagName, err := services.GetLatestReleaseFromGitHub("mikoto2000", "port-forwarder")
			if err != nil {
				return "", err
			}

			pattern := "pattern"
			var tmpl *template.Template
			if containerArch == "amd64" {
				tmpl, err = template.New(pattern).Parse(downloadURLPortForwarderContainerAmd64Pattern)
			} else if containerArch == "aarch64" {
				tmpl, err = template.New(pattern).Parse(downloadURLPortForwarderContainerArm64Pattern)
			} else {
				return "", errors.New("port-forwarder-container download error: Unknown arch")
			}
			if err != nil {
				return "", err
			}

			tmplParams := map[string]string{"TagName": latestTagName}
			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				panic(err)
			}
			return downloadURL.String(), nil
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: download,
	}
}
