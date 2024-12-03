package tools

import (
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// ホスト上で起動する port-forwarder のツール情報
var PortForwarderHost Tool = Tool{
	FileName: PortForwarderHostFileName,
	CalculateDownloadURL: func(containerArch string) (string, error) {
		latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "port-forwarder")
		if err != nil {
			return "", err
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(downloadURLPortForwarderCliPattern)
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
	installFunc: func(downloadURL string, filePath string, containerArch string) (string, error) {
		return simpleInstall(downloadURL, filePath)
	},
}

// コンテナ上で起動する port-forwarder のツール情報
var PortForwarderContainer Tool = Tool{
	FileName: "port-forwarder",
	CalculateDownloadURL: func(containerArch string) (string, error) {
		// TODO: containerArch に応じてダウンロードするファイルを切り替える
		latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "port-forwarder")
		if err != nil {
			return "", err
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(downloadURLPortForwarderCliPattern)
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
	installFunc: func(downloadURL string, filePath string, containerArch string) (string, error) {
		return simpleInstall(downloadURL, filePath)
	},
}

