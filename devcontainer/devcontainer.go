package devcontainer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/dockerCompose"
)

const CONTAINER_COMMAND = "docker"

var DEVCONTAINRE_ARGS_PREFIX = []string{"up"}

func ExecuteDevcontainer(args []string, devcontainerFilePath string, vimFilePath string) {
	vimFileName := filepath.Base(vimFilePath)

	// `devcontainer up` でコンテナを起動

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// 末尾以外のものはそのまま `devcontainer up` への引数として渡す
	userArgs := args[0 : len(args)-1]
	userArgs = append(userArgs, "--workspace-folder", workspaceFolder)
	devcontainerArgs := append(DEVCONTAINRE_ARGS_PREFIX, userArgs...)
	fmt.Printf("run container: `%s \"%s\"`\n", devcontainerFilePath, strings.Join(devcontainerArgs, "\" \""))
	dockerRunCommand := exec.Command(devcontainerFilePath, devcontainerArgs...)

	stderrByteBuffer := &bytes.Buffer{}
	dockerRunCommand.Stderr = stderrByteBuffer

	stdout, err := dockerRunCommand.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, stderrByteBuffer.String())
		panic(err)
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	containerId := upCommandResult.ContainerId
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

	dockerVimArgs := []string{"exec", "--container-id", containerId, "--workspace-folder", workspaceFolder, "/" + vimFileName, "--appimage-extract-and-run"}
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerFilePath, strings.Join(dockerVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, devcontainerFilePath, dockerVimArgs...)
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

	// コンテナ停止は別途 down コマンドで行う
}

func Down(args []string, devcontainerFilePath string) {

	// `devcontainer read-configuration` で docker compose の利用判定

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]
	readConfigurationArgs := []string{"read-configuration", "--workspace-folder", workspaceFolder}
	fmt.Printf("run devcontainer: `%s %s\n", devcontainerFilePath, strings.Join(readConfigurationArgs, " "))
	devconteinerReadConfigurationCommand := exec.Command(devcontainerFilePath, readConfigurationArgs...)

	stdout, _ := devconteinerReadConfigurationCommand.Output()
	stdoutString := string(stdout)

	if stdoutString == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		os.Exit(0)
	}

	// `dockerComposeFile` が含まれているかを確認する
	// 含まれているなら docker compose によるコンテナ構築がされている
	if strings.Contains(stdoutString, "dockerComposeFile") {

		// docker compose ps コマンドで compose の情報取得
		dockerComposePsResultString, err := dockerCompose.Ps(workspaceFolder)
		if err != nil {
			panic(err)
		}
		if dockerComposePsResultString == "" {
			fmt.Println("devcontainer already downed.")
			os.Exit(0)
		}

		// docker compose ps コマンドの結果からプロジェクト名を取得
		projectName, err := dockerCompose.GetProjectName(dockerComposePsResultString)
		if err != nil {
			panic(err)
		}

		// プロジェクト名を使って docker compose down を実行
		fmt.Printf("Run `docker compose -p %s down`(Async)\n", projectName)
		err = dockerCompose.Down(projectName)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("single container down not implements...")
	}

}

