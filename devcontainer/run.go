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
)

var devcontainerRunArgsPrefix = []string{"run", "-d", "--rm"}
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
func Run(args []string, vimFilePath string, cdrPath string, configDirForDocker string, vimrc string, defaultRunargs []string) error {
	vimFileName := filepath.Base(vimFilePath)

	// バックグラウンドでコンテナを起動
	// `docker run -d --rm os.Args[1:] sh -c "sleep infinity"`
	devcontainerRunArgs := devcontainerRunArgsPrefix
	// windows でなければ、 runargs を使用する
	if runtime.GOOS != "windows" {
		devcontainerRunArgs = append(devcontainerRunArgs, defaultRunargs...)
	}
	devcontainerRunArgs = append(devcontainerRunArgs, args...)
	devcontainerRunArgs = append(devcontainerRunArgs, devcontainerRunArgsSuffix...)
	fmt.Printf("run container: `%s \"%s\"`\n", containerCommand, strings.Join(devcontainerRunArgs, "\" \""))
	dockerRunCommand := exec.Command(containerCommand, devcontainerRunArgs...)
	containerIDRaw, err := dockerRunCommand.CombinedOutput()
	containerID := string(containerIDRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerID))
		return &ContainerStartError{msg: "Container start error."}
	}
	containerID = strings.ReplaceAll(containerID, "\n", "")
	containerID = strings.ReplaceAll(containerID, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerID)

	// clipboard-data-receiver を起動
	configDirForCdr := filepath.Join(configDirForDocker, containerID)
	err = os.MkdirAll(configDirForCdr, 0744)
	if err != nil {
		return err
	}
	pid, port, err := tools.RunCdr(cdrPath, configDirForCdr)
	if err != nil {
		return err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
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
		return &ChmodError{msg: "chmod error."}
	}
	fmt.Printf(" done.\n")

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDocker, port)
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

	dockerRunVimArgs := dockerRunVimArgs(containerID, vimFileName)
	fmt.Printf("Start vim: `%s \"%s\"`\n", containerCommand, strings.Join(dockerRunVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, containerCommand, dockerRunVimArgs...)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	// 失敗してもコンテナのあと片付けはしたいのでエラーを無視
	dockerExec.Run()

	// コンテナ停止
	// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
	fmt.Printf("Stop container(Async) %s.\n", containerID)
	err = exec.Command(containerCommand, "stop", containerID).Start()
	if err != nil {
		return err
	}

	// clipboard-data-receiver を停止
	err = tools.KillCdr(pid)
	if err != nil {
		return err
	}
	err = os.RemoveAll(configDirForCdr)
	if err != nil {
		return err
	}

	return nil
}
