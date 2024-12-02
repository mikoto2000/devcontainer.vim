package devcontainer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

var devcontainreArgsPrefix = []string{"up"}

// devcontainer でコンテナを立ち上げ、 Vim を転送し、実行する。
// 既存実装の都合上、configFilePath から configDirForDevcontainer を抽出している
func Start(args []string, devcontainerPath string, vimFilePath string, cdrPath, configFilePath string, vimrc string) error {

	vimFileName := filepath.Base(vimFilePath)

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// `devcontainer up` でコンテナを起動

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
		return err
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		return err
	}
	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	// clipboard-data-receiver を起動
	configDirForDevcontainer := filepath.Dir(configFilePath)
	pid, port, err := tools.RunCdr(cdrPath, configDirForDevcontainer)
	if err != nil {
		return err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	containerID := upCommandResult.ContainerID

	err = docker.Cp("vim", vimFilePath, containerID, "/")
	if err != nil {
		return err
	}

	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
	dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		return err
	}
	fmt.Printf(" done.\n")

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDevcontainer, port)
	if err != nil {
		return err
	}

	// コンテナへ SendToTcp.vim を転送
	err = docker.Cp("SendToTcp.vim", sendToTCP, containerID, "/")
	if err != nil {
		return err
	}

	// コンテナへ vimrc を転送
	err = docker.Cp("vimrc", vimrc, containerID, "/")
	if err != nil {
		return err
	}

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	devcontainerStartVimArgs := devcontainerStartVimArgs(containerID, workspaceFolder, vimFileName)
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerStartVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, devcontainerPath, devcontainerStartVimArgs...)
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

	// コンテナ停止は別途 down コマンドで行う
	return nil
}
