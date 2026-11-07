package tools

import (
	"fmt"
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

			var tmpl *template.Template
			if containerArch == "amd64" {
				tmpl, err = template.New("port-forwarder-amd64-download-url").Parse(downloadURLPortForwarderContainerAmd64Pattern)
			} else if containerArch == "aarch64" {
				tmpl, err = template.New("port-forwarder-arm64-download-url").Parse(downloadURLPortForwarderContainerArm64Pattern)
			} else {
				return "", fmt.Errorf("port-forwarder-container download error: Unknown Architecture: %q", containerArch)
			}
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
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: services.Download,
	}
}
