package devcontainer

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

type TestDevcontainerStartUseService struct{}

func (s TestDevcontainerStartUseService) StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, sendToTCP string, containerArch string, useSystemVim bool) error {
	return nil
}

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

	args := []string{"../test/project/TestStart"}

	// devcontainer を用いたコンテナ立ち上げ
	err = Start(TestDevcontainerStartUseService{}, args, devcontainerPath, cdrPath, binDir, nvim, configFilePath, "../test/resource/TestStart/vimrc")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error executing devcontainer: %v\n", err)
		}
		os.Exit(1)
	}

	// 後片付け
	defer Down([]string{"../test/project/TestStart"}, devcontainerPath, configDirForDevcontainer)

	// json マージ後の設定でコンテナが起動するか？
	// 起動したコンテナに所望のファイルが転送されているか？
	//     ストレージのマウントがされるか
	vimfilesOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "ls -d ~/.vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimfilesWant := "/home/vscode/.vim"
	if !strings.Contains(vimfilesOutput, vimfilesWant) {
		t.Fatalf("error: want match %s, but got %s", vimfilesWant, vimfilesOutput)
	}
	//     portForward がされるか
	//     TODO: なぜかテストでは生えてこない...
	//pfOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "ls /pf")
	//if err != nil {
	//	t.Fatalf("error: %s", err)
	//}
	//pfWant := "localhost:8888_"
	//if !strings.Contains(pfOutput, pfWant) {
	//	t.Fatalf("error: want match %s, but got %s", pfWant, pfOutput)
	//}

	//     環境変数が設定されるか
	termOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "\"env\"")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	termWantMatch := "TERM=xterm-256color"
	if !strings.Contains(termOutput, termWantMatch) {
		t.Fatalf("error: want match %s, but got %s", termWantMatch, termOutput)
	}
	//     /vim
	vimOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "ls /vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
	}
	//     /vimrc
	vimrcOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
	//     /port-forwarder
	portForwarderOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", "../test/project/TestStart", "sh", "-c", "ls /port-forwarder")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	portForwarderWant := "port-forwarder"
	if !strings.Contains(portForwarderOutput, portForwarderWant) {
		t.Fatalf("error: want match %s, but got %s", portForwarderWant, portForwarderOutput)
	}
}

func TestStartWithDockerCompose(t *testing.T) {
	// TODO: chdir しなくても成功するように修正
	os.Chdir("../test/project/TestStartWithDockerCompose")
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
	configFilePath, err := CreateConfigFile(devcontainerPath, ".", configDirForDevcontainer)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		}
		os.Exit(1)
	}

	args := []string{"."}

	// devcontainer を用いたコンテナ立ち上げ
	err = Start(TestDevcontainerStartUseService{}, args, devcontainerPath, cdrPath, binDir, nvim, configFilePath, "../../resource/TestStartWithDockerCompose/vimrc")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error executing devcontainer: %v\n", err)
		}
		os.Exit(1)
	}

	// 後片付け
	defer Down([]string{"."}, devcontainerPath, configDirForDevcontainer)

	// json マージ後の設定でコンテナが起動するか？
	// 起動したコンテナに所望のファイルが転送されているか？
	//     ストレージのマウントがされるか
	vimfilesOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "ls -d ~/.vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimfilesWant := "/home/vscode/.vim"
	if !strings.Contains(vimfilesOutput, vimfilesWant) {
		t.Fatalf("error: want match %s, but got %s", vimfilesWant, vimfilesOutput)
	}

	//     環境変数が設定されるか
	termOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "\"env\"")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	termWantMatch := "TERM=xterm-256color"
	if !strings.Contains(termOutput, termWantMatch) {
		t.Fatalf("error: want match %s, but got %s", termWantMatch, termOutput)
	}
	//     /vim
	vimOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "ls /vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
	}
	//     /vimrc
	vimrcOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
	//     /port-forwarder
	portForwarderOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "ls /port-forwarder")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	portForwarderWant := "port-forwarder"
	if !strings.Contains(portForwarderOutput, portForwarderWant) {
		t.Fatalf("error: want match %s, but got %s", portForwarderWant, portForwarderOutput)
	}

}
