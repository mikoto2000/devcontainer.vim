package tools

import (
	"fmt"
	"runtime"
)

// devcontainer/cli のツール情報
var DEVCONTAINER = func(services InstallerUseServices) Tool {
	return devcontainerToolForHost(services, runtime.GOOS, runtime.GOARCH)
}

func devcontainerToolForHost(services InstallerUseServices, hostOS, hostArch string) Tool {
	const owner, repository = "mikoto2000", "devcontainers-cli"
	const baseName = "devcontainer"
	suffix := ""
	assetArch := hostArch
	if hostOS == "windows" {
		suffix = ".exe"
		assetArch = "x64"
	} else if assetArch == "amd64" {
		assetArch = "x64"
	}

	return Tool{
		FileName: baseName + suffix,
		CalculateDownloadURL: func(_ string) (string, error) {
			switch hostOS {
			case "darwin", "linux", "windows":
			default:
				return "", fmt.Errorf("unsupported Dev Container CLI host OS: %q", hostOS)
			}
			tag, err := services.GetLatestReleaseFromGitHub(owner, repository)
			if err != nil {
				return "", err
			}
			asset := fmt.Sprintf("%s-%s-%s-%s%s", baseName, hostOS, assetArch, tag, suffix)
			return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repository, tag, asset), nil
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: services.Download,
	}
}
