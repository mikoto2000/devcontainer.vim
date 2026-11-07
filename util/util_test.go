package util

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
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

	defer func() {
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

	defer func() {
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

	defer func() {
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
	defer func() {
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

func TestCreateConfigFileForDevcontainer(t *testing.T) {
	configDirForDevcontainer := t.TempDir()
	workspaceFolder := filepath.Join("test", "resource", "TestCreateConfigFileForDevcontainer")
	configDir := filepath.Join(workspaceFolder, ".devcontainer")
	configFilePath := filepath.Join(configDir, "devcontainer.json")
	additionalConfigFilePath := filepath.Join(configDir, "devcontainer.vim.json")

	mergedConfigFilePath, err := CreateConfigFileForDevcontainer(configDirForDevcontainer, workspaceFolder, configFilePath, additionalConfigFilePath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	data, err := os.ReadFile(mergedConfigFilePath)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	var merged map[string]string
	if err := json.Unmarshal(data, &merged); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"name":           "test_name",
		"image":          "test_image",
		"additional_key": "additional_value",
	} {
		if got := merged[key]; got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestGetConfigDir(t *testing.T) {
	t.Chdir(t.TempDir())
	configDir := filepath.Join("cache", "config")
	workspaceFolder := filepath.Join("project", "workspace with spaces")
	workspaceAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		t.Fatal(err)
	}
	hash := md5.Sum([]byte(workspaceAbs))
	want := filepath.Join(configDir, fmt.Sprintf("%x", hash))
	for _, input := range []string{workspaceFolder, workspaceAbs} {
		got, err := GetConfigDir(configDir, input)
		if err != nil || got != want {
			t.Errorf("GetConfigDir(%q, %q) = %q, %v, want %q", configDir, input, got, err, want)
		}
	}
}

func TestIsWsl(t *testing.T) {
	os.Setenv("WSL_DISTRO_NAME", "")
	got := IsWsl()
	if got != true {
		t.Fatalf("error: want true, but got false")
	}

	os.Unsetenv("WSL_DISTRO_NAME")
	got = IsWsl()
	if got != false {
		t.Fatalf("error: want false, but got true")
	}
}

func TestCreateFileWithContents(t *testing.T) {
	file := "test/TestCreateFileWithContents"
	contents := "testing"

	err := CreateFileWithContents(file, contents, 0755)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	defer os.RemoveAll(file)

	// ファイルの存在確認
	fileInfo, err := os.Stat(file)
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	// モードが設定した通りか？
	wantFileModeString := "-rwxr-xr-x"
	gotFileModeString := fileInfo.Mode().String()
	if wantFileModeString != gotFileModeString {
		t.Fatalf("error: want %s, but got %s", wantFileModeString, gotFileModeString)
	}

	// コンテンツが設定した通りか？
	bytes, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if contents != string(bytes) {
		t.Fatalf("error: want %s, but got %s", contents, string(bytes))
	}

}
