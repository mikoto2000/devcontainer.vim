package tools

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
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
				return "", errors.New("Unknown Architecture")
			}

			latestTagName, err := service.GetLatestReleaseFromGitHub("tmux", "tmux-builds")
			if err != nil {
				return "", err
			}

			pattern := "pattern"
			tmpl, err := template.New(pattern).Parse(tmuxDownloadURLPattern)
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

			err = os.Chmod(filePath, 0755)
			if err != nil {
				return filePath, err
			}
			return filePath, nil
		},
		DownloadFunc: download,
	}
}

func extractSingleFileFromTarGz(archivePath string, targetBaseName string, destPath string) error {
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
	for {
		hdr, err := tr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("target file not found in archive")
			}
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != targetBaseName {
			continue
		}

		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, tr)
		return err
	}
}
