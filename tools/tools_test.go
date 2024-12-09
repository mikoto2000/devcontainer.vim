package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

type TestDownloadService struct{}

func (s TestDownloadService) Download(_ string, destPath string) error {
	return os.WriteFile(destPath, []byte{}, 0755)
}

func TestInstallStartTools(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, cdrPath, err := InstallStartTools(binDir)
	if err != nil {
		t.Fatalf("Error installing start tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "devcontainer")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	// cdr の存在確認
	wantCdrPath := filepath.Join(binDir, "clipboard-data-receiver")
	if wantCdrPath != cdrPath {
		t.Fatalf("want %s, but got %s", wantCdrPath, cdrPath)
	}
	_, err = os.Stat(wantCdrPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallRunTools(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, err := InstallRunTools(binDir)
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "clipboard-data-receiver")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallVimAmd64(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, err := InstallVim(binDir, false, "amd64")
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "vim_amd64")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallVimAarch64(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, err := InstallVim(binDir, false, "aarch64")
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "vim_aarch64")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallNVimAmd64(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, err := InstallVim(binDir, true, "amd64")
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "nvim_amd64")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallNVimAarch64(t *testing.T) {
	GetDownloadService = func() DownloadService {
		tds := TestDownloadService{}
		return tds
	}

	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, err := InstallVim(binDir, true, "aarch64")
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, "nvim_aarch64")
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
