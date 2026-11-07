package tools

import (
	"fmt"
	"slices"
	"testing"
)

func TestVimDownloadURL(t *testing.T) {
	const (
		staticRepo    = "mikoto2000/vim-static"
		appImageRepo  = "vim/vim-appimage"
		amd64Static   = "https://github.com/mikoto2000/vim-static/releases/download/v9.1.1353/vim-v9.1.1353-x86_64.tar.gz"
		arm64Static   = "https://github.com/mikoto2000/vim-static/releases/download/v9.1.1353/vim-v9.1.1353-aarch64.tar.gz"
		amd64AppImage = "https://github.com/vim/vim-appimage/releases/download/v9.1.9999/Vim-v9.1.9999.glibc2.34-x86_64.AppImage"
	)
	for _, tt := range []struct {
		hostOS, arch, repository, url string
	}{
		{"darwin", "amd64", staticRepo, amd64Static},
		{"darwin", "x86_64", staticRepo, amd64Static},
		{"linux", "amd64", appImageRepo, amd64AppImage},
		{"darwin", "arm64", staticRepo, arm64Static},
		{"linux", "aarch64", staticRepo, arm64Static},
	} {
		t.Run(tt.hostOS+"/"+tt.arch, func(t *testing.T) {
			service := &vimReleaseTestService{}
			got, err := vimDownloadURL(service, tt.hostOS, tt.arch)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(service.lookups, []string{tt.repository}) {
				t.Errorf("release lookups = %v, want [%s]", service.lookups, tt.repository)
			}
			if got != tt.url {
				t.Errorf("download URL = %q, want %q", got, tt.url)
			}
		})
	}
}

type vimReleaseTestService struct {
	TestInstallerUseServices
	lookups []string
}

func (s *vimReleaseTestService) GetLatestReleaseFromGitHub(owner, repository string) (string, error) {
	repo := owner + "/" + repository
	s.lookups = append(s.lookups, repo)
	switch repo {
	case "mikoto2000/vim-static":
		return "v9.1.1353", nil
	case "vim/vim-appimage":
		return "v9.1.9999", nil
	default:
		return "", fmt.Errorf("unexpected release lookup for %s", repo)
	}
}
