package devcontainer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

var devcontainerRunArgsPrefix = []string{"run", "-d", "--rm", "--add-host=host.docker.internal:host-gateway"}
var devcontainerRunArgsSuffix = []string{"sh", "-c", "trap \"exit 0\" TERM; sleep infinity & wait"}

type ContainerStartError struct {
	msg string
}

func (e *ContainerStartError) Error() string {
	return e.msg
}

type ChmodError struct {
	msg string
}

func (e *ChmodError) Error() string {
	return e.msg
}

// docker run で、ワンショットでコンテナを立ち上げる
func Run(
	args []string,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	shell string,
	configDirForDocker string,
	vimrc string,
	defaultRunargs []string) error {

	// コンテナのセットアップ
	containerID, vimFileName, sendToTCP, containerArch, useSystemVim, cdrPid, cdrConfigDir, err := setupContainer(
		args,
		cdrPath,
		vimInstallDir,
		nvim,
		configDirForDocker,
		vimrc,
		defaultRunargs)

	// 後片付け
	// clipboard-data-receiver を停止
	defer func() {
		err = tools.KillCdr(cdrPid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}

		err = os.RemoveAll(cdrConfigDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cache remove error: %s\n", err)
		}
	}()

	// コンテナ停止
	defer func() {
		// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
		fmt.Printf("Stop container(Async) %s.\n", containerID)
		err = exec.Command(containerCommand, "stop", containerID).Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}
	}()

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sendToTCPName := filepath.Base(sendToTCP)
	dockerRunVimArgs := dockerRunVimArgs(containerID, vimFileName, sendToTCPName, containerArch, useSystemVim, shell, configDirForDocker)
	fmt.Printf("Start vim: `%s \"%s\"`\n", containerCommand, strings.Join(dockerRunVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, containerCommand, dockerRunVimArgs...)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	err = dockerExec.Run()
	if err != nil {
		return err
	}

	return nil
}

// コンテナを起動し、コンテナIDを返す
func startContainer(args []string, defaultRunargs []string) (string, error) {
	devcontainerRunArgs := devcontainerRunArgsPrefix
	// windows でなければ、 runargs を使用する
	if runtime.GOOS != "windows" {
		devcontainerRunArgs = append(devcontainerRunArgs, defaultRunargs...)
	}
	devcontainerRunArgs = append(devcontainerRunArgs, args...)
	devcontainerRunArgs = append(devcontainerRunArgs, devcontainerRunArgsSuffix...)
	fmt.Printf("run container: `%s \"%s\"`\n", containerCommand, strings.Join(devcontainerRunArgs, "\" \""))

	dockerRunCommand := exec.Command(containerCommand, devcontainerRunArgs...)
	containerIDRaw, err := dockerRunCommand.Output()
	containerID := string(containerIDRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerID))
		return "", &ContainerStartError{msg: "Container start error."}
	}

	containerID = strings.ReplaceAll(containerID, "\n", "")
	containerID = strings.ReplaceAll(containerID, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerID)

	return containerID, nil
}

// コンテナのアーキテクチャを取得する
func getContainerArch(containerID string) (string, error) {
	containerArch, err := docker.Exec(containerID, "uname", "-m")
	if err != nil {
		return "", err
	}
	containerArch = strings.TrimSpace(containerArch)
	containerArch, err = util.NormalizeContainerArch(containerArch)
	if err != nil {
		return "", err
	}
	fmt.Printf("Container Arch: '%s'.\n", containerArch)
	return containerArch, nil
}

// port-forwarderをコンテナにインストールする
func installPortForwarder(containerID, vimInstallDir, containerArch string) error {
	portForwarderContainerPath, err := tools.PortForwarderContainer(tools.DefaultInstallerUseServices{}).Install(vimInstallDir, containerArch, false)
	if err != nil {
		return err
	}
	err = docker.Cp("port-forwarder-container", portForwarderContainerPath, containerID, "/port-forwarder")
	if err != nil {
		return err
	}
	return nil
}

// clipboard-data-receiverを起動する
func startClipboardReceiver(cdrPath, configDirForDocker, containerID string) (int, int, string, error) {
	configDirForCdr := filepath.Join(configDirForDocker, containerID)
	err := os.MkdirAll(configDirForCdr, 0744)
	if err != nil {
		return 0, 0, configDirForCdr, err
	}
	pid, port, err := tools.RunCdr(cdrPath, configDirForCdr)
	if err != nil {
		return 0, 0, configDirForCdr, err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)
	return pid, port, configDirForCdr, nil
}

// Vimの検出とインストールを行う
func setupVim(containerID, vimInstallDir string, nvim bool, containerArch string) (string, bool, error) {
	vimFileName := "vim"
	if nvim {
		vimFileName = "nvim"
	}

	useSystemVim := false
	fmt.Printf("Check system installed %s ... ", vimFileName)
	out, _ := docker.Exec(containerID, "which", vimFileName)
	if out != "" {
		fmt.Printf("found.\n")
		useSystemVim = true

		if nvim {
			vimFileName = "nvim"
		}
	} else {
		fmt.Printf("not found.\n")

		if runtime.GOARCH == "arm64" {
			// arm の場合スタティックリンクの nvim を作れないため、 vim にフォールバック
			vimFileName = "vim"
			nvim = false
		} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" && nvim {
			// M1 Mac で amd64 のコンテナを動かすと、なぜか AppImage が動かないので vim にフォールバック
			vimFileName = "vim"
			nvim = false
		}
	}
	fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))

	if !useSystemVim {
		// コンテナへ Vim/Neovim を転送して実行権限を追加
		vimFilePath, err := tools.InstallVim(vimInstallDir, nvim, containerArch)
		if err != nil {
			return "", false, err
		}

		err = docker.Cp("vim", vimFilePath, containerID, "/"+vimFileName)
		if err != nil {
			return vimFileName, useSystemVim, err
		}

		// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
		dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
		fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
		chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
		if err != nil {
			fmt.Fprintln(os.Stderr, "chmod error.")
			fmt.Fprintln(os.Stderr, string(chmodResult))
			return vimFileName, useSystemVim, &ChmodError{msg: "chmod error."}
		}
		fmt.Printf(" done.\n")
	}

	return vimFileName, useSystemVim, nil
}

// Vimファイル（SendToTcp.vimとvimrc）をコンテナに転送する
func transferVimFiles(containerID, configDirForDocker, vimrc string, port int, isNvim bool) (string, error) {
	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDocker, port, isNvim)
	if err != nil {
		return "", err
	}

	// コンテナへ SendToTcp.vim を転送
	err = docker.Cp("SendToTcp.vim", sendToTCP, containerID, "/")
	if err != nil {
		return sendToTCP, err
	}

	// コンテナへ vimrc を転送
	err = docker.Cp("vimrc", vimrc, containerID, "/")
	if err != nil {
		return sendToTCP, err
	}

	return sendToTCP, nil
}

func setupContainer(
	args []string,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	configDirForDocker string,
	vimrc string,
	defaultRunargs []string) (string, string, string, string, bool, int, string, error) {

	// 1. コンテナを起動
	containerID, err := startContainer(args, defaultRunargs)
	if err != nil {
		return "", "", "", "", false, 0, "", err
	}

	// 2. コンテナアーキテクチャを取得
	containerArch, err := getContainerArch(containerID)
	if err != nil {
		return containerID, "", "", "", false, 0, "", err
	}

	// 3. port-forwarderをインストール
	err = installPortForwarder(containerID, vimInstallDir, containerArch)
	if err != nil {
		return containerID, "", "", containerArch, false, 0, "", err
	}

	// 4. clipboard-data-receiverを起動
	pid, port, configDirForCdr, err := startClipboardReceiver(cdrPath, configDirForDocker, containerID)
	if err != nil {
		return containerID, "", "", containerArch, false, pid, configDirForCdr, err
	}

	// 5. Vimの検出とインストール
	vimFileName, useSystemVim, err := setupVim(containerID, vimInstallDir, nvim, containerArch)
	if err != nil {
		return containerID, vimFileName, "", containerArch, useSystemVim, pid, configDirForCdr, err
	}

	// 6. Vimファイルを転送
	sendToTCP, err := transferVimFiles(containerID, configDirForDocker, vimrc, port, vimFileName == "nvim")
	if err != nil {
		return containerID, vimFileName, sendToTCP, containerArch, useSystemVim, pid, configDirForCdr, err
	}

	return containerID, vimFileName, sendToTCP, containerArch, useSystemVim, pid, configDirForCdr, nil
}
