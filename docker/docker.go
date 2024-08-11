package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/tools"
)

const containerCommand = "docker"

var dockerRunArgsPrefix = []string{"run", "-d", "--rm"}
var dockerRunArgsSuffix = []string{"sh", "-c", "trap \"exit 0\" TERM; sleep infinity & wait"}

func Run(args []string, vimFilePath string, cdrPath string, configDirForDocker string, vimrc string, defaultRunargs []string) {
	vimFileName := filepath.Base(vimFilePath)

	// バックグラウンドでコンテナを起動
	// `docker run -d --rm os.Args[1:] sh -c "sleep infinity"`
	dockerRunArgs := dockerRunArgsPrefix
	// windows でなければ、 runargs を使用する
	if runtime.GOOS != "windows" {
		dockerRunArgs = append(dockerRunArgs, defaultRunargs...)
	}
	dockerRunArgs = append(dockerRunArgs, args...)
	dockerRunArgs = append(dockerRunArgs, dockerRunArgsSuffix...)
	fmt.Printf("run container: `%s \"%s\"`\n", containerCommand, strings.Join(dockerRunArgs, "\" \""))
	dockerRunCommand := exec.Command(containerCommand, dockerRunArgs...)
	containerIDRaw, err := dockerRunCommand.CombinedOutput()
	containerID := string(containerIDRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerID))
		panic(err)
	}
	containerID = strings.ReplaceAll(containerID, "\n", "")
	containerID = strings.ReplaceAll(containerID, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerID)

	// clipboard-data-receiver を起動
	configDirForCdr := filepath.Join(configDirForDocker, containerID)
	err = os.MkdirAll(configDirForCdr, 0744)
	if err != nil {
		panic(err)
	}
	pid, port, err := tools.RunCdr(cdrPath, configDirForCdr)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	err = Cp("AppImage", vimFilePath, containerID, "/")
	if err != nil {
		panic(err)
	}

	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
	dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDocker, port)
	if err != nil {
		panic(err)
	}

	// コンテナへ SendToTcp.vim を転送
	err = Cp("SendToTcp.vim", sendToTCP, containerID, "/")
	if err != nil {
		panic(err)
	}

	// コンテナへ vimrc を転送
	err = Cp("vimrc", vimrc, containerID, "/")
	if err != nil {
		panic(err)
	}

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dockerVimArgs := []string{
		"exec",
		"-it",
		containerID,
		"sh",
		"-c",
		"cd ~; /" + vimFileName + " --appimage-extract > /dev/null; cd -; ~/squashfs-root/AppRun -S /SendToTcp.vim -S /vimrc",
	}
	fmt.Printf("Start vim: `%s \"%s\"`\n", containerCommand, strings.Join(dockerVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, containerCommand, dockerVimArgs...)
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
		panic(err)
	}

	// clipboard-data-receiver を停止
	tools.KillCdr(pid)
	err = os.RemoveAll(configDirForCdr)
	if err != nil {
		panic(err)
	}
}

// workspaceFolder で指定したディレクトリに対応するコンテナのコンテナ ID を返却する
func GetContainerIDFromWorkspaceFolder(workspaceFolder string) (string, error) {

	// `devcontainer.local_folder=${workspaceFolder}` が含まれている行を探す

	workspaceFilderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return "", err
	}

	psResult, err := Ps("label=devcontainer.local_folder=" + workspaceFilderAbs)
	if err != nil {
		return "", err
	}

	id, err := GetID(psResult)
	if err != nil {
		return "", err
	}

	return id, nil
}

// `docker ps --format json` コマンドを実行する。
func Ps(filter string) (string, error) {
	dockerPsCommand := exec.Command("docker", "ps", "--format", "json", "--filter", filter)
	stdout, err := dockerPsCommand.Output()
	return string(stdout), err
}

// `docker stop -f ${containerID}` コマンドを実行する。
func Stop(containerID string) error {
	dockerStopCommand := exec.Command("docker", "stop", containerID)
	err := dockerStopCommand.Start()
	return err
}

// `docker rm -f ${containerID}` コマンドを実行する。
func Rm(containerID string) error {
	dockerRmCommand := exec.Command("docker", "rm", "-f", containerID)
	err := dockerRmCommand.Start()
	return err
}

func Cp(tagForLog string, from string, containerID string, to string) error {
	dockerCpArgs := []string{"cp", from, containerID + ":" + to}
	fmt.Printf("Copy %s: `%s \"%s\"` ...", tagForLog, containerCommand, strings.Join(dockerCpArgs, "\" \""))
	copyResult, err := exec.Command(containerCommand, dockerCpArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "AppImage copy error.")
		fmt.Fprintln(os.Stderr, string(copyResult))
		return err
	}
	fmt.Printf(" done.\n")
	return nil
}
