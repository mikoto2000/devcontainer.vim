package devcontainer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/docker"
	"github.com/mikoto2000/devcontainer.vim/dockerCompose"
	"github.com/mikoto2000/devcontainer.vim/tools"
	"github.com/mikoto2000/devcontainer.vim/util"
)

const CONTAINER_COMMAND = "docker"

var DEVCONTAINRE_ARGS_PREFIX = []string{"up"}

// devcontainer でコンテナを立ち上げ、 Vim を転送し、実行する。
// 既存実装の都合上、configFilePath から configDirForDevcontainer を抽出している
func ExecuteDevcontainer(args []string, devcontainerPath string, vimFilePath string, cdrPath, configFilePath string, vimrc string) {
	vimFileName := filepath.Base(vimFilePath)

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// `devcontainer up` でコンテナを起動

	// 末尾以外のものはそのまま `devcontainer up` への引数として渡す
	userArgs := args[0 : len(args)-1]
	userArgs = append(userArgs, "--override-config", configFilePath, "--workspace-folder", workspaceFolder)
	devcontainerArgs := append(DEVCONTAINRE_ARGS_PREFIX, userArgs...)
	fmt.Printf("run container: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerArgs, "\" \""))
	dockerRunCommand := exec.Command(devcontainerPath, devcontainerArgs...)
	dockerRunCommand.Stderr = os.Stderr

	stdout, err := dockerRunCommand.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		panic(err)
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		panic(err)
	}
	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	// clipboard-data-receiver を起動
	configDirForDevcontainer := filepath.Dir(configFilePath)
	pid, port, err := tools.RunCdr(cdrPath, configDirForDevcontainer)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

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
	dockerChownArgs := []string{"exec", "--user", "root", containerId, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", CONTAINER_COMMAND, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(CONTAINER_COMMAND, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTcp, err := tools.CreateSendToTcp(configDirForDevcontainer, port)
	if err != nil {
		panic(err)
	}

	// コンテナへ SendToTcp.vim を転送
	err = docker.Cp("SendToTcp.vim", sendToTcp, containerId, "/")
	if err != nil {
		panic(err)
	}

	// コンテナへ vimrc を転送
	err = docker.Cp("vimrc", vimrc, containerId, "/")
	if err != nil {
		panic(err)
	}

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dockerVimArgs := []string{
		"exec",
		"--container-id",
		containerId,
		"--workspace-folder",
		workspaceFolder,
		"sh",
		"-c",
		"/" + vimFileName + "--appimage-extract; ./squashfs-root/AppRun -S /SendToTcp.vim -S /vimrc"}
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerPath, strings.Join(dockerVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, devcontainerPath, dockerVimArgs...)
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

func Down(args []string, devcontainerPath string, configDirForDevcontainer string) {

	// `devcontainer read-configuration` で docker compose の利用判定

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]
	stdout, _ := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder)
	if stdout == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		os.Exit(0)
	}

	// `dockerComposeFile` が含まれているかを確認する
	// 含まれているなら docker compose によるコンテナ構築がされている
	var configDir string
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

		// pid ファイル参照のために、
		// コンテナ別の設定ファイル格納ディレクトリの名前(コンテナIDを記録)を記録
		configDir = util.GetConfigDir(configDirForDevcontainer, workspaceFolder)
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

		// pid ファイル参照のために、
		// コンテナ別の設定ファイル格納ディレクトリの名前(コンテナIDを記録)を記録
		configDir = util.GetConfigDir(configDirForDevcontainer, workspaceFolder)
	}

	// clipboard-data-receiver を停止
	pidFile := filepath.Join(configDir, "pid")
	fmt.Printf("Read PID file: %s\n", pidFile)
	pidStringBytes, err := os.ReadFile(pidFile)
	if err != nil {
		panic(err)
	}
	pid, err := strconv.Atoi(string(pidStringBytes))
	if err != nil {
		panic(err)
	}
	fmt.Printf("clipboard-data-receiver PID: %d\n", pid)
	tools.KillCdr(pid)

	err = os.RemoveAll(configDir)
	if err != nil {
		panic(err)
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

func Templates(devcontainerFilePath string, templatesArgs ...string) (string, error) {
	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := templatesArgs[len(templatesArgs)-1]
	otherArgs := templatesArgs[:len(templatesArgs)-1]

	args := append([]string{"templates"}, otherArgs...)
	args = append(args, "--workspace-folder", workspaceFolder)
	return ExecuteCombineOutput(devcontainerFilePath, args...)
}

func Execute(devcontainerFilePath string, args ...string) (string, error) {
	fmt.Printf("run devcontainer: `%s %s`\n", devcontainerFilePath, strings.Join(args, " "))
	cmd := exec.Command(devcontainerFilePath, args...)
	stdout, err := cmd.Output()
	return string(stdout), err
}

func ExecuteCombineOutput(devcontainerFilePath string, args ...string) (string, error) {
	fmt.Printf("run devcontainer: `%s %s`\n", devcontainerFilePath, strings.Join(args, " "))
	cmd := exec.Command(devcontainerFilePath, args...)
	stdout, err := cmd.CombinedOutput()
	return string(stdout), err
}
