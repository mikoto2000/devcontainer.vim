package devcontainer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

var devcontainerRunArgsPrefix = []string{"run", "-d", "--rm", "--add-host=host.docker.internal:host-gateway"}
var devcontainerRunArgsSuffix = []string{"sh", "-c", "trap \"exit 0\" TERM; sleep infinity & wait"}

type ContainerStartError struct {
	msg string
	err error
}

func (e *ContainerStartError) Error() string {
	return e.msg
}

func (e *ContainerStartError) Unwrap() error {
	return e.err
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
	noCdr bool,
	noPf bool,
	noTmux bool,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	shell string,
	configDirForDocker string,
	vimrc string,
	defaultRunargs []string) error {

	// コンテナのセットアップ
	setup, err := setupContainer(
		args,
		noCdr,
		noPf,
		noTmux,
		cdrPath,
		vimInstallDir,
		nvim,
		configDirForDocker,
		vimrc,
		defaultRunargs)

	// セットアップが途中で失敗しても、取得済みのリソースを片付ける。
	defer func() {
		if setup.cdrPID > 0 {
			if err := tools.KillCdr(setup.cdrPID); err != nil {
				fmt.Fprintf(os.Stderr, "Clipboard receiver stop error: %s\n", err)
			}
		}
		if setup.cdrConfigDir != "" {
			if err := os.RemoveAll(setup.cdrConfigDir); err != nil {
				fmt.Fprintf(os.Stderr, "Cache remove error: %s\n", err)
			}
		}
		if setup.containerID != "" {
			fmt.Printf("Stop container %s.\n", setup.containerID)
			output, err := exec.Command(containerCommand, "stop", setup.containerID).CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Container stop error: %v\n%s", err, output)
			}
		}
	}()
	if err != nil {
		return err
	}

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sendToTCPName := filepath.Base(setup.sendToTCP)
	dockerRunVimArgs, err := dockerRunVimArgs(setup.containerID, setup.vimFileName, setup.tmuxFileName, sendToTCPName,
		setup.containerArch, setup.useSystemVim, setup.useSystemTmux, noTmux, shell, configDirForDocker)
	if err != nil {
		return err
	}
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
	if err != nil {
		diagnostic := string(containerIDRaw)
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			diagnostic += "\n" + string(exitErr.Stderr)
		}
		msg := fmt.Sprintf("docker run: %v", err)
		if diagnostic := strings.TrimSpace(diagnostic); diagnostic != "" {
			msg += ": " + diagnostic
		}
		return "", &ContainerStartError{msg: msg, err: err}
	}

	containerID := strings.TrimSpace(string(containerIDRaw))
	fmt.Printf("Container started. id: %s\n", containerID)

	return containerID, nil
}

// clipboard-data-receiverを起動する
func startClipboardReceiver(cdrPath, configDirForDocker, containerID string) (int, int, string, error) {
	configDirForCdr := filepath.Join(configDirForDocker, containerID)
	err := os.MkdirAll(configDirForCdr, 0744)
	if err != nil {
		return 0, 0, "", err
	}
	pid, port, err := tools.RunCdr(cdrPath, configDirForCdr)
	if err != nil {
		return pid, 0, configDirForCdr, err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)
	return pid, port, configDirForCdr, nil
}

// containerSetup also records resources acquired before a setup failure.
type containerSetup struct {
	containerID   string
	containerArch string
	cdrPID        int
	cdrConfigDir  string
	vimFileName   string
	useSystemVim  bool
	tmuxFileName  string
	useSystemTmux bool
	sendToTCP     string
}

func setupContainer(
	args []string,
	noCdr bool,
	noPf bool,
	noTmux bool,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	configDirForDocker string,
	vimrc string,
	defaultRunargs []string) (containerSetup, error) {
	var setup containerSetup
	var err error

	// 1. コンテナを起動
	setup.containerID, err = startContainer(args, defaultRunargs)
	if err != nil {
		return setup, err
	}

	// 2. コンテナアーキテクチャを取得
	setup.containerArch, err = getContainerArch(setup.containerID)
	if err != nil {
		return setup, err
	}

	// 3. port-forwarderをインストール
	if !noPf {
		if err := installPortForwarder(setup.containerID, vimInstallDir, setup.containerArch); err != nil {
			return setup, err
		}
	}

	// 4. clipboard-data-receiverを起動
	port := 0
	if !noCdr {
		setup.cdrPID, port, setup.cdrConfigDir, err = startClipboardReceiver(cdrPath, configDirForDocker, setup.containerID)
		if err != nil {
			return setup, err
		}
	}

	// 5. Vimの検出とインストール
	setup.vimFileName, setup.useSystemVim, err = setupVim(setup.containerID, vimInstallDir, nvim, setup.containerArch)
	if err != nil {
		return setup, err
	}

	if !noTmux {
		setup.tmuxFileName, setup.useSystemTmux, err = setupTmux(setup.containerID, vimInstallDir, setup.containerArch)
		if err != nil {
			return setup, err
		}
	}

	// 6. Vimファイルを転送
	setup.sendToTCP, err = transferVimFiles(setup.containerID, configDirForDocker, vimrc, noCdr, port, setup.vimFileName == "nvim")
	return setup, err
}
