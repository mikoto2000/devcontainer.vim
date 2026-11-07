package devcontainer

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

type TestDevcontainerStartUseService struct{}

func (s TestDevcontainerStartUseService) StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePathForDevcontainer string) error {
	return nil
}

func TestStart(t *testing.T) {
	requireLinuxDocker(t)
	_, binDir, _, configDirForDevcontainer := createTempAppDirs(t)

	// 必要なファイルのダウンロード
	devcontainerPath := requireTestBinary(t, "devcontainer")
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	// devcontainerコマンドが動作するかテスト
	if err := exec.CommandContext(t.Context(), devcontainerPath, "--version").Run(); err != nil {
		t.Skipf("devcontainer CLI not working: %v", err)
	}

	workspace := copyStartTestWorkspace(t, "TestStart")
	configFilePath, err := CreateConfigFile(devcontainerPath, workspace, configDirForDevcontainer)
	if err != nil {
		t.Fatalf("Error creating config file: %v", err)
	}

	startTestWorkspace(t, workspace, devcontainerPath, cdrPath, binDir, configFilePath, configDirForDevcontainer, "../test/resource/TestStart/vimrc")

	// json マージ後の設定でコンテナが起動するか？
	// 起動したコンテナに所望のファイルが転送されているか？
	//     ストレージのマウントがされるか
	vimfilesOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls -d ~/.vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimfilesWant := "/home/vscode/.vim"
	if !strings.Contains(vimfilesOutput, vimfilesWant) {
		t.Fatalf("error: want match %s, but got %s", vimfilesWant, vimfilesOutput)
	}
	//     環境変数が設定されるか
	termOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "\"env\"")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	termWantMatch := "TERM=xterm-256color"
	if !strings.Contains(termOutput, termWantMatch) {
		t.Fatalf("error: want match %s, but got %s", termWantMatch, termOutput)
	}
	//     /vim
	vimOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
	}
	//     /vimrc
	vimrcOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
	//     /port-forwarder
	portForwarderOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /port-forwarder")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	portForwarderWant := "port-forwarder"
	if !strings.Contains(portForwarderOutput, portForwarderWant) {
		t.Fatalf("error: want match %s, but got %s", portForwarderWant, portForwarderOutput)
	}
}

func TestStartWithDockerCompose(t *testing.T) {
	requireLinuxDocker(t)
	_, binDir, _, configDirForDevcontainer := createTempAppDirs(t)

	// 必要なファイルのダウンロード
	devcontainerPath := requireTestBinary(t, "devcontainer")
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	workspace := copyStartTestWorkspace(t, "TestStartWithDockerCompose")
	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	configFilePath, err := CreateConfigFile(devcontainerPath, workspace, configDirForDevcontainer)
	if err != nil {
		t.Fatalf("Error creating config file: %v", err)
	}

	startTestWorkspace(t, workspace, devcontainerPath, cdrPath, binDir, configFilePath, configDirForDevcontainer, "../test/resource/TestStartWithDockerCompose/vimrc")

	// json マージ後の設定でコンテナが起動するか？
	// 起動したコンテナに所望のファイルが転送されているか？
	//     ストレージのマウントがされるか
	vimfilesOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls -d ~/.vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimfilesWant := "/home/vscode/.vim"
	if !strings.Contains(vimfilesOutput, vimfilesWant) {
		t.Fatalf("error: want match %s, but got %s", vimfilesWant, vimfilesOutput)
	}

	//     環境変数が設定されるか
	termOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "\"env\"")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	termWantMatch := "TERM=xterm-256color"
	if !strings.Contains(termOutput, termWantMatch) {
		t.Fatalf("error: want match %s, but got %s", termWantMatch, termOutput)
	}
	//     /vim
	vimOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /vim")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
	}
	//     /vimrc
	vimrcOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
	//     /port-forwarder
	portForwarderOutput, err := Execute(devcontainerPath, "exec", "--workspace-folder", workspace, "sh", "-c", "ls /port-forwarder")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	portForwarderWant := "port-forwarder"
	if !strings.Contains(portForwarderOutput, portForwarderWant) {
		t.Fatalf("error: want match %s, but got %s", portForwarderWant, portForwarderOutput)
	}

}

func copyStartTestWorkspace(t *testing.T, fixture string) string {
	t.Helper()
	workspace := t.TempDir()
	if err := os.CopyFS(workspace, os.DirFS(filepath.Join("..", "test", "project", fixture))); err != nil {
		t.Fatal(err)
	}
	// Bind only test fixtures, including when the caller has real SSH keys.
	home := t.TempDir()
	for _, name := range []string{".vim", ".ssh"} {
		if err := os.Mkdir(filepath.Join(home, name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, ".gitconfig"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	return workspace
}

func startTestWorkspace(t *testing.T, workspace, devcontainerPath, cdrPath, binDir, configFilePath, cacheDir, vimrc string) {
	t.Helper()
	args := []string{workspace}
	startErr := Start(TestDevcontainerStartUseService{}, args, devcontainerPath, false, false, false, cdrPath, binDir, false, "", configFilePath, vimrc)
	t.Cleanup(func() {
		err := Down(args, devcontainerPath, cacheDir)
		var notFound *docker.ContainerNotFoundError
		if err != nil && (startErr == nil || !errors.As(err, &notFound)) {
			t.Errorf("remove devcontainer test workspace: %v", err)
		}
	})
	if startErr != nil {
		t.Fatalf("Error executing devcontainer: %v", startErr)
	}
}
