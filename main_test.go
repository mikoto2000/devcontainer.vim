package main

import (
	"os"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func TestCreateConfigFile(t *testing.T) {
	defer os.RemoveAll("./test/resource")
	_, binDir, _, configDirForDevcontainer, err := util.CreateCacheDirectory(func() (string, error) {
		return "./test", nil
	}, "resource")
	if err != nil {
		panic(err)
	}

	devcontainerPath, _, err := tools.InstallStartTools(binDir)
	if err != nil {
		t.Fatalf("Error installing start tools: %v", err)
	}

	workspaceFolder := "./test/project/TestCreateConfigFile"
	configFilePath, err := createConfigFile(devcontainerPath, workspaceFolder, configDirForDevcontainer)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := "test/resource/config/devcontainer/6d0900e89898e089cf294371661aea37/devcontainer.json"
	if want != configFilePath {
		t.Fatalf("error: want %s, but got %s", want, configFilePath)
	}
}
