package tools

import (
	"fmt"
	"runtime"
)

// Vim のツール情報
var VIM = func(service InstallerUseServices) Tool {
	return Tool{
		FileName: "vim",
		CalculateDownloadURL: func(containerArch string) (string, error) {
			return vimDownloadURL(service, runtime.GOOS, containerArch)
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: service.Download,
	}
}

func vimDownloadURL(service InstallerUseServices, hostOS, containerArch string) (string, error) {
	owner, repository := "mikoto2000", "vim-static"
	var assetPattern string
	switch containerArch {
	case "amd64", "x86_64":
		if hostOS == "darwin" {
			assetPattern = "vim-%s-x86_64.tar.gz"
		} else {
			owner, repository = "vim", "vim-appimage"
			assetPattern = "Vim-%s.glibc2.34-x86_64.AppImage"
		}
	case "arm64", "aarch64":
		assetPattern = "vim-%s-aarch64.tar.gz"
	default:
		return "", fmt.Errorf("Unknown Architecture: %q", containerArch)
	}

	tag, err := service.GetLatestReleaseFromGitHub(owner, repository)
	if err != nil {
		return "", err
	}
	assetName := fmt.Sprintf(assetPattern, tag)
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s", owner, repository, tag, assetName), nil
}
