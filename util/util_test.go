package util

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
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

func TestGetConfigDirectorySuccess(t *testing.T) {
	base := t.TempDir()
	want := filepath.Join(base, "success")
	_, err := CreateConfigDirectory(func() (string, error) { return base, nil }, "success")
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	_, err = os.Stat(want)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestGetConfigDirectoryFailed(t *testing.T) {
	base := t.TempDir()
	wantErr := errors.New("base directory lookup failed")
	got, err := CreateConfigDirectory(func() (string, error) { return base, wantErr }, "failed")
	if got != "" || !errors.Is(err, wantErr) {
		t.Fatalf("CreateConfigDirectory = %q, %v, want an empty path and %v", got, err, wantErr)
	}
	if _, err := os.Stat(filepath.Join(base, "failed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("directory created after lookup failure: %v", err)
	}
}

func TestCreateCacheDirectorySuccess(t *testing.T) {
	base := t.TempDir()
	wantBase := filepath.Join(base, "success")
	cache, bin, dockerConfig, devcontainerConfig, err := CreateCacheDirectory(func() (string, error) { return base, nil }, "success")
	if err != nil {
		t.Fatalf("error: %s", err)
	}

	for _, tt := range []struct{ name, got, want string }{
		{"cache", cache, wantBase},
		{"bin", bin, filepath.Join(wantBase, "bin")},
		{"Docker configuration", dockerConfig, filepath.Join(wantBase, "config", "docker")},
		{"Dev Container configuration", devcontainerConfig, filepath.Join(wantBase, "config", "devcontainer")},
	} {
		if tt.got != tt.want {
			t.Errorf("%s path = %q, want %q", tt.name, tt.got, tt.want)
			continue
		}
		info, err := os.Stat(tt.got)
		if err != nil {
			t.Error(err)
		} else if !info.IsDir() {
			t.Errorf("%s path %q is not a directory", tt.name, tt.got)
		}
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

func TestCreateConfigFileForDevcontainerMergesConflicts(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "devcontainer.json")
	additional := filepath.Join(dir, "devcontainer.vim.json")
	if err := os.WriteFile(base, []byte(`{
		"name": "base",
		"mounts": ["base-mount"],
		"remoteEnv": {"PATH": "/base/bin", "KEEP": "base"},
		"postCreateCommand": ["old", "argument"],
		"init": true,
	}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(additional, []byte(`{
		// Settings specific to this editor.
		"name": "vim",
		"mounts": ["vim-mount"],
		"remoteEnv": {"PATH": "/vim/bin", "ADD": "vim"},
		"postCreateCommand": "new command",
		"init": false
	}`), 0600); err != nil {
		t.Fatal(err)
	}

	path, err := CreateConfigFileForDevcontainer(filepath.Join(dir, "cache"), dir, base, additional)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name":              "vim",
		"mounts":            []any{"base-mount", "vim-mount"},
		"remoteEnv":         map[string]any{"PATH": "/vim/bin", "KEEP": "base", "ADD": "vim"},
		"postCreateCommand": "new command",
		"init":              false,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("merged configuration = %s, want %v", data, want)
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
	t.Setenv("WSL_DISTRO_NAME", "")
	got := IsWsl()
	if got != true {
		t.Fatalf("error: want true, but got false")
	}

	if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
		t.Fatal(err)
	}
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
