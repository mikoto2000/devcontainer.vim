package util

import (
	"testing"

	"os"
	"path/filepath"
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

func TestCreateDirectorySuccess(t *testing.T) {

	// Create directory
	// TempDir にテスト用ディレクトリを作成
	createdDirectory := CreateDirectory(CreateTestCreateDirectorySuccessPath, "TestCreateDirectory")

	if !IsExists(createdDirectory) {
		t.Fatalf("directory create failed: %s.", createdDirectory)
	}

	// 後片付け
	os.RemoveAll(filepath.Dir(createdDirectory))
}

func CreateTestCreateDirectoryFailedPath() (string, error) {
	return "/usr/bin", nil
}

func TestCreateDirectoryFailed(t *testing.T) {
	// panic したらテスト失敗
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("ディレクトリ作成失敗したのに panic しませんでした。")
		}
	}()

	// Create directory
	// 失敗させるために必ず存在するであろう /usr/bin/sh に作ろうとする
	CreateDirectory(CreateTestCreateDirectoryFailedPath, "sh")
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
