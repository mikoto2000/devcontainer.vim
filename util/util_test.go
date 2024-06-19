package util

import (
	"testing"

	"os"
)

func TestIsExistsCommandOk(t *testing.T) {
	got := IsExistsCommand("ls")
	want := true
	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}
}

func TestIsExistsCommandNg(t *testing.T) {
	got := IsExistsCommand("noExistsCommand")
	want := false
	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}
}

func CreateTestCreateDirectorySuccessPath() (string, error) {
	tempDir := os.TempDir()
	testDirectory, err := os.MkdirTemp(tempDir, "devcontainer.vim-testdir")
	return testDirectory, err
}

func CreateTestCreateDirectoryFailedPath() (string, error) {
	return "/usr/bin", nil
}

func TestIsExistsTrue(t *testing.T) {
	want := true

	// 存在するファイルの存在確認
	got := IsExists("util_test.go")

	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}

}

func TestIsExistsFalse(t *testing.T) {
	want := false

	// 存在しないファイルの存在確認
	got := IsExists("noExistsFile")

	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}

}
