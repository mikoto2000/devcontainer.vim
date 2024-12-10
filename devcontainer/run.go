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
func Run(
	args []string,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	configDirForDocker string,
	vimrc string,
	defaultRunargs []string) error {

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
	containerIDRaw, err := dockerRunCommand.Output()
	containerID := string(containerIDRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerID))
		return &ContainerStartError{msg: "Container start error."}
	}
	containerID = strings.ReplaceAll(containerID, "\n", "")
	containerID = strings.ReplaceAll(containerID, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerID)

	// コンテナ停止
	defer func() {
		// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
		fmt.Printf("Stop container(Async) %s.\n", containerID)
		err = exec.Command(containerCommand, "stop", containerID).Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}
	}()

	// コンテナ内に入り、コンテナの Arch を確認
	containerArch, err := docker.Exec(containerID, "uname", "-m")
	if err != nil {
		return err
	}
	containerArch = strings.TrimSpace(containerArch)
	containerArch, err = util.NormalizeContainerArch(containerArch)
	if err != nil {
		return err
	}
	fmt.Printf("Container Arch: '%s'.\n", containerArch)

	vimFilePath, err := tools.InstallVim(vimInstallDir, nvim, containerArch)
	if err != nil {
		return err
	}

	portForwarderContainerPath, err := tools.PortForwarderContainer(tools.DefaultInstallerUseServices{}).Install(vimInstallDir, containerArch, false)
	if err != nil {
		return err
	}
	err = docker.Cp("port-forwarder-container", portForwarderContainerPath, containerID, "/port-forwarder")
	if err != nil {
		return err
	}

	vimFileName := filepath.Base(vimFilePath)

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

	// clipboard-data-receiver を停止
	defer func() {
		err = tools.KillCdr(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}

		err = os.RemoveAll(configDirForCdr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cache remove error: %s\n", err)
		}
	}()

	useSystemVim := false
	fmt.Printf("Check system installed %s ... ", vimFileName)
	out, _ := docker.Exec(containerID, "which", vimFileName)
	if out != "" {
		fmt.Printf("found.\n")
		useSystemVim = true
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
		// コンテナへ appimage を転送して実行権限を追加
		// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`

		err = docker.Cp("vim", vimFilePath, containerID, "/"+vimFileName)
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
	}

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDocker, port, vimFileName == "nvim")
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

	sendToTCPName := filepath.Base(sendToTCP)
	dockerRunVimArgs := dockerRunVimArgs(containerID, vimFileName, sendToTCPName, containerArch, useSystemVim)
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
	err = dockerExec.Run()
	if err != nil {
		return err
	}

	return nil
}
