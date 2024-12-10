package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

type TestInstallerUseServices struct{}

func (s TestInstallerUseServices) GetLatestReleaseFromGitHub(owner string, repository string) (string, error) {
	return "", nil
}

func (s TestInstallerUseServices) Download(downloadURL string, destPath string) error {
	os.WriteFile(destPath, []byte{}, 0755)
	return nil
}

func TestInstallStartTools(t *testing.T) {
	defer os.RemoveAll("test")
	_, binDir, _, _, err := util.CreateCacheDirectory(func() (string, error) {
		return "test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, cdrPath, err := InstallStartTools(TestInstallerUseServices{}, binDir)
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

func TestSelfUpdate(t *testing.T) {
	err := SelfUpdate(TestInstallerUseServices{})
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}
