package dockerRun

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

const CONTAINER_COMMAND = "docker"

var DOCKER_RUN_ARGS_PREFIX = []string{"run", "-d", "--rm"}
var DOCKER_RUN_ARGS_SUFFIX = []string{"sh", "-c", "trap \"exit 0\" TERM; sleep infinity & wait"}

func ExecuteDockerRun(args []string, vimFilePath string, vimFileName string) {
	// バックグラウンドでコンテナを起動
	// `docker run -d --rm os.Args[1:] sh -c "sleep infinity"`
	dockerRunArgs := append(DOCKER_RUN_ARGS_PREFIX, args...)
	dockerRunArgs = append(dockerRunArgs, DOCKER_RUN_ARGS_SUFFIX...)
	fmt.Printf("run container: `%s \"%s\"`\n", CONTAINER_COMMAND, strings.Join(dockerRunArgs, "\" \""))
	dockerRunCommand := exec.Command(CONTAINER_COMMAND, dockerRunArgs...)
	containerIdRaw, err := dockerRunCommand.CombinedOutput()
	containerId := string(containerIdRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerId))
		panic(err)
	}
	containerId = strings.ReplaceAll(containerId, "\n", "")
	containerId = strings.ReplaceAll(containerId, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerId)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	dockerCpArgs := []string{"cp", vimFilePath, containerId + ":/"}
	fmt.Printf("Copy AppImage: `%s \"%s\"` ...", CONTAINER_COMMAND, strings.Join(dockerCpArgs, "\" \""))
	copyResult, err := exec.Command(CONTAINER_COMMAND, dockerCpArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "AppImage copy error.")
		fmt.Fprintln(os.Stderr, string(copyResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
	dockerChownArgs := []string{"exec", containerId, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", CONTAINER_COMMAND, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(CONTAINER_COMMAND, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dockerVimArgs := []string{"exec", "-it", containerId, "/" + vimFileName, "--appimage-extract-and-run"}
	fmt.Printf("Start vim: `%s \"%s\"`", CONTAINER_COMMAND, strings.Join(dockerVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, CONTAINER_COMMAND, dockerVimArgs...)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	err = dockerExec.Run()
	if err != nil {
		panic(err)
	}

	// コンテナ停止
	// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
	fmt.Printf("Stop container(Async) %s.\n", containerId)
	err = exec.Command(CONTAINER_COMMAND, "stop", containerId).Start()
	if err != nil {
		panic(err)
	}
}

