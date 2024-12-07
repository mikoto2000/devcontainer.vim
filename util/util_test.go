package util

import (
	"encoding/json"
	"errors"
	"path/filepath"
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

func pathFuncHello() (string, error) {
	return "Hello", nil
}

func pathFuncFailed() (string, error) {
	return "", errors.New("failed!")
}

func TestGetConfigDirectorySuccess(t *testing.T) {
	want := "Hello/success"
	// 存在しないファイルの存在確認
	got, err := CreateConfigDirectory(pathFuncHello, "success")
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	defer func () {
		os.RemoveAll(filepath.Dir(got))
	}()

	_, err = os.Stat(want)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestGetConfigDirectoryFailed(t *testing.T) {
	want := "Hello/failed"
	// 存在しないファイルの存在確認
	got, err := CreateConfigDirectory(pathFuncFailed, "failed")
	if err == nil {
		t.Fatalf("not return error got: %s", got)
	}

	defer func () {
		os.RemoveAll(filepath.Dir(got))
	}()

	_, err = os.Stat(want)
	if err == nil {
		t.Fatalf("not return error found: %s", got)
	}
}

func TestCreateCacheDirectorySuccess(t *testing.T) {
	wantBase := "Hello/success"
	// 存在しないファイルの存在確認
	gotAppCacheDir, _, _, _, err := CreateCacheDirectory(pathFuncHello, "success")
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	defer func () {
		os.RemoveAll(filepath.Dir(gotAppCacheDir))
	}()

	_, err = os.Stat(filepath.Join(wantBase))
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	_, err = os.Stat(filepath.Join(wantBase, "bin"))
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	_, err = os.Stat(filepath.Join(wantBase, "config", "docker"))
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	_, err = os.Stat(filepath.Join(wantBase, "config", "devcontainer"))
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestAddExecutePermission(t *testing.T) {
	target := "TestAddExecutePermission"
	err := os.Mkdir(target, 0644)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	defer func () {
		os.RemoveAll(target)
	}()

	err = AddExecutePermission(target)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	fileInfo, err := os.Stat(target)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	want := "drwxr-xr-x"
	got := fileInfo.Mode().String()
	if got != want {
		t.Fatalf("error: want %s but got %s", got, want)
	}
}

func TestParseJwcc(t *testing.T) {
	target := "test/resource/TestParseJwcc.json"
	jsonBytes, err := ParseJwcc(target)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	var unmarshaledJson map[string]interface{}
	json.Unmarshal(jsonBytes, &unmarshaledJson)

	want := "test_value"
	got := unmarshaledJson["test_key"]

	if got != want {
		t.Fatalf("error: want %s but got %s", got, want)
	}
}
