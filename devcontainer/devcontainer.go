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

	"github.com/mikoto2000/devcontainer.vim/docker"
	"github.com/mikoto2000/devcontainer.vim/dockerCompose"
	"github.com/mikoto2000/devcontainer.vim/util"
)

const CONTAINER_COMMAND = "docker"

var DEVCONTAINRE_ARGS_PREFIX = []string{"up"}

func ExecuteDevcontainer(args []string, devcontainerFilePath string, vimFilePath string, configFilePath string) {
	vimFileName := filepath.Base(vimFilePath)

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// `devcontainer up` でコンテナを起動

	// 末尾以外のものはそのまま `devcontainer up` への引数として渡す
	userArgs := args[0 : len(args)-1]
	userArgs = append(userArgs, "--config", configFilePath, "--workspace-folder", workspaceFolder)
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
	stdout, _ := ReadConfiguration(devcontainerFilePath, "--workspace-folder", workspaceFolder)
	if stdout == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		os.Exit(0)
	}

	// `dockerComposeFile` が含まれているかを確認する
	// 含まれているなら docker compose によるコンテナ構築がされている
	if strings.Contains(stdout, "dockerComposeFile") {

		// docker compose ps コマンドで compose の情報取得
		dockerComposePsResultString, err := dockerCompose.Ps(workspaceFolder)
		if err != nil {
			panic(err)
		}
		if dockerComposePsResultString == "" {
			fmt.Println("devcontainer already downed.")
			os.Exit(0)
		}

		// 必要なのは最初の 1 行だけなので、最初の 1 行のみを取得
		dockerComposePsResultFirstItemString := strings.Split(dockerComposePsResultString, "\n")[0]

		// docker compose ps コマンドの結果からプロジェクト名を取得
		projectName, err := dockerCompose.GetProjectName(dockerComposePsResultFirstItemString)
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
		// ワークスペースに対応するコンテナを探して ID を取得する
		containerId, err := docker.GetContainerIdFromWorkspaceFolder(workspaceFolder)
		if err != nil {
			panic(err)
		}

		// 取得したコンテナに対して rm を行う
		fmt.Printf("Run `docker rm -f %s down`(Async)\n", containerId)
		err = docker.Rm(containerId)
		if err != nil {
			panic(err)
		}
	}
}

func GetConfigurationFilePath(devcontainerFilePath string, workspaceFolder string) (string, error) {
	stdout, _ := ReadConfiguration(devcontainerFilePath, "--workspace-folder", workspaceFolder)
	return GetConfigFilePath(stdout)
}

func ReadConfiguration(devcontainerFilePath string, readConfiguration ...string) (string, error) {
	args := append([]string{"read-configuration"}, readConfiguration...)
	return Execute(devcontainerFilePath, args...)
}

func Execute(devcontainerFilePath string, args ...string) (string, error) {
	fmt.Printf("run devcontainer: `%s %s`\n", devcontainerFilePath, strings.Join(args, " "))
	cmd := exec.Command(devcontainerFilePath, args...)
	stdout, err := cmd.Output()
	return string(stdout), err
}

// devcontainer.vim 用の追加設定ファイルを探す。
// bool: 追加設定ファイルの有無(true: 有, false: 無)
// string: 追加設定ファイルのパス
func FindAdditionalConfiguration(configFilePath string) (bool, string, error) {
	// configurationFilePath と同じ階層に同じ名前で拡張子が `.vim.json` であるものを探す
	configurationFileName := configFilePath[:len(configFilePath)-len(filepath.Ext(configFilePath))]
	additionalConfigurationFilePath := configurationFileName + ".vim.json"
	return util.IsExists(additionalConfigurationFilePath), additionalConfigurationFilePath, nil
}
