package tools

import (
	"fmt"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// devcontainer/cli のツール情報
var DEVCONTAINER Tool = Tool{
	FileName: DEVCONTAINER_FILE_NAME,
	CalculateDownloadUrl: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "devcontainers-cli")
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf(DOWNLOAD_URL_DEVCONTAINERS_CLI_PATTERN, latestTagName, latestTagName)
	},
	installFunc: func(downloadUrl string, installDir string, fileName string, override bool) (string, error) {
		return simpleInstall(downloadUrl, installDir, fileName, override)
	},
}

