package devcontainer

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func TestStart(t *testing.T) {
	appName := "devcontainer.vim"
	_, err := util.CreateConfigDirectory(os.UserConfigDir, appName)
	if err != nil {
		panic(err)
	}
	_, binDir, _, configDirForDevcontainer, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		panic(err)
	}

	// 必要なファイルのダウンロード
	nvim := false
	devcontainerPath, cdrPath, err := tools.InstallStartTools(tools.DefaultInstallerUseServices{}, binDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing start tools: %v\n", err)
		os.Exit(1)
	}

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	configFilePath, err := CreateConfigFile(devcontainerPath, "../test/project/TestStart", configDirForDevcontainer)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		}
		os.Exit(1)
	}

	// devcontainer を用いたコンテナ立ち上げ
	err = Start([]string{"../test/project/TestStart"}, devcontainerPath, cdrPath, binDir, nvim, configFilePath, "../test/resource/TestStart/vimrc")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error executing devcontainer: %v\n", err)
		}
		os.Exit(1)
	}
}
