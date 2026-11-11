package devcontainer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/host"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

type DevcontainerStartUseService interface {
	StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePathForDevcontainer string) error
}

type DefaultDevcontainerStartUseService struct{}

func (s DefaultDevcontainerStartUseService) StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configDirForDevcontainer string) error {
	return startVim(containerID, devcontainerPath, workspaceFolder, vimFileName, tmuxFileName, sendToTCP, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configDirForDevcontainer)
}

var devcontainreArgsPrefix = []string{"up"}

// devcontainer up でコンテナを起動し、コンテナIDを返す
func startDevcontainer(devcontainerPath string, args []string, configFilePath string, workspaceFolder string) (string, error) {
	// 末尾以外のものはそのまま `devcontainer up` への引数として渡す
	userArgs := args[0 : len(args)-1]
	userArgs = append(userArgs, "--override-config", configFilePath, "--workspace-folder", workspaceFolder)
	devcontainerArgs := append(devcontainreArgsPrefix, userArgs...)
	fmt.Printf("run container: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerArgs, "\" \""))

	dockerRunCommand := exec.Command(devcontainerPath, devcontainerArgs...)
	dockerRunCommand.Stderr = os.Stderr

	stdout, err := dockerRunCommand.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		return "", err
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		return "", err
	}

	containerID := upCommandResult.ContainerID
	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	return containerID, nil
}

// devcontainer用のclipboard-data-receiverを起動する
func startClipboardReceiverForDevcontainer(cdrPath, configDirForDevcontainer string) (int, int, error) {
	pid, port, err := tools.RunCdr(cdrPath, configDirForDevcontainer)
	if err != nil {
		return 0, 0, err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)
	return pid, port, nil
}

// devcontainer でコンテナを立ち上げ、 Vim を転送し、実行する。
// 既存実装の都合上、configFilePath から configDirForDevcontainer を抽出している
func Start(
	services DevcontainerStartUseService,
	args []string,
	devcontainerPath string,
	noCdr bool,
	noPf bool,
	noTmux bool,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	shell string,
	configFilePath string,
	vimrc string) error {

	if len(args) == 0 {
		return errors.New("missing workspace folder")
	}

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// 1. devcontainer up でコンテナを起動
	containerID, err := startDevcontainer(devcontainerPath, args, configFilePath, workspaceFolder)
	if err != nil {
		return err
	}

	// 2. コンテナアーキテクチャを取得
	containerArch, err := getContainerArch(containerID)
	if err != nil {
		return err
	}

	// 3. port-forwarderをインストール
	if !noPf {
		if err := installPortForwarder(containerID, vimInstallDir, containerArch); err != nil {
			return err
		}
	}

	// 4. clipboard-data-receiverを起動
	port := 0
	configDirForDevcontainer := filepath.Dir(configFilePath)
	if !noCdr {
		_, port, err = startClipboardReceiverForDevcontainer(cdrPath, configDirForDevcontainer)
		if err != nil {
			return err
		}
	}

	// 5. port-forwardingの設定
	if !noPf {
		stopForwarding, err := setupPortForwarding(context.Background(), containerID, devcontainerPath, workspaceFolder, configFilePath)
		if err != nil {
			return err
		}
		defer stopForwarding()
	}

	// 6. Vimの検出とインストール
	vimFileName, useSystemVim, err := setupVim(containerID, vimInstallDir, nvim, containerArch)
	if err != nil {
		return err
	}

	tmuxFileName := ""
	useSystemTmux := false
	if !noTmux {
		tmuxFileName, useSystemTmux, err = setupTmux(containerID, vimInstallDir, containerArch)
		if err != nil {
			return err
		}
	}

	// 7. Vimファイルの転送
	sendToTCP, err := transferVimFiles(containerID, configDirForDevcontainer, vimrc, noCdr, port, vimFileName == "nvim")
	if err != nil {
		return err
	}

	// 8. コンテナへ接続
	err = services.StartVim(containerID, devcontainerPath, workspaceFolder, vimFileName, tmuxFileName, sendToTCP, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configDirForDevcontainer)
	if err != nil {
		return err
	}

	// コンテナ停止は別途 down コマンドで行う
	return nil
}

// コンテナへ接続
// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`
func startVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePathForDevcontainer string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sendToTCPName := filepath.Base(sendToTCP)
	devcontainerStartVimArgs, err := devcontainerStartVimArgs(containerID, workspaceFolder, vimFileName, tmuxFileName, sendToTCPName, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configFilePathForDevcontainer)
	if err != nil {
		return err
	}
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerStartVimArgs, "\" \""))
	dockerExec := createStartVimCommand(ctx, devcontainerPath, devcontainerStartVimArgs)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	return dockerExec.Run()
}

func createStartVimCommand(ctx context.Context, devcontainerPath string, devcontainerStartVimArgs []string) *exec.Cmd {
	if host.IsWsl() && host.IsExistsCommand("script") {
		scriptArgs := []string{"-qefc", shellQuote(devcontainerPath)}
		for _, arg := range devcontainerStartVimArgs {
			scriptArgs[1] += " " + shellQuote(arg)
		}
		scriptArgs = append(scriptArgs, "/dev/null")
		return exec.CommandContext(ctx, "script", scriptArgs...)
	}

	return exec.CommandContext(ctx, devcontainerPath, devcontainerStartVimArgs...)
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
