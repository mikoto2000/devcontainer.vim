package tools

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/host"
)

const tmuxDownloadURLPattern = "https://github.com/tmux/tmux-builds/releases/download/{{ .TagName }}/tmux-{{ .Version }}-{{ .Platform }}.tar.gz"

var Tmux = func(service InstallerUseServices) Tool {
	return Tool{
		FileName: "tmux",
		CalculateDownloadURL: func(containerArch string) (string, error) {
			var platform string
			switch containerArch {
			case "amd64", "x86_64":
				platform = "linux-x86_64"
			case "arm64", "aarch64":
				platform = "linux-arm64"
			default:
				return "", fmt.Errorf("Unknown Architecture: %q", containerArch)
			}

			latestTagName, err := service.GetLatestReleaseFromGitHub("tmux", "tmux-builds")
			if err != nil {
				return "", err
			}

			tmpl, err := template.New("tmux-download-url").Parse(tmuxDownloadURLPattern)
			if err != nil {
				return "", err
			}

			tmplParams := map[string]string{
				"TagName":  latestTagName,
				"Version":  strings.TrimPrefix(latestTagName, "v"),
				"Platform": platform,
			}

			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				return "", err
			}
			return downloadURL.String(), nil
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			archivePath := filePath + ".tar.gz"
			err := downloadFunc(downloadURL, archivePath)
			if err != nil {
				return filePath, err
			}
			defer os.Remove(archivePath)

			err = extractSingleFileFromTarGz(archivePath, "tmux", filePath)
			if err != nil {
				return filePath, err
			}

			err = host.AddExecutePermission(filePath)
			if err != nil {
				return filePath, err
			}
			return filePath, nil
		},
		DownloadFunc: service.Download,
	}
}

func extractSingleFileFromTarGz(archivePath, targetName, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	found := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg || strings.TrimPrefix(hdr.Name, "./") != targetName {
			continue
		}
		if found {
			return errors.New("multiple target files found at archive root")
		}

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, tr)
		if err := errors.Join(copyErr, out.Close()); err != nil {
			return err
		}
		found = true
	}
	if !found {
		return errors.New("target file not found at archive root")
	}
	return nil
}
