package devcontainer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/docker"
	"github.com/mikoto2000/devcontainer.vim/dockercompose"
	"github.com/mikoto2000/devcontainer.vim/tools"
	"github.com/mikoto2000/devcontainer.vim/util"
)

const containerCommand = "docker"

var devcontainreArgsPrefix = []string{"up"}

type UnknownTypeError struct {
	msg string
}

func (e *UnknownTypeError) Error() string {
	return e.msg
}

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

	dockerVimArgs := []string{
		"exec",
		"--container-id",
		containerID,
		"--workspace-folder",
		workspaceFolder,
		"sh",
		"-c",
		"cd ~; /" + vimFileName + " --appimage-extract > /dev/null; cd -; ~/squashfs-root/AppRun --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
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
		return err
	}

	// コンテナ停止は別途 down コマンドで行う
	return nil
}

func Stop(args []string, devcontainerPath string, configDirForDevcontainer string) error {

	// `devcontainer read-configuration` で docker compose の利用判定

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]
	stdout, _ := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder)
	if stdout == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		return nil
	}

	// `dockerComposeFile` が含まれているかを確認する
	// 含まれているなら docker compose によるコンテナ構築がされている
	if strings.Contains(stdout, "dockerComposeFile") {

		// docker compose ps コマンドで compose の情報取得
		dockerComposePsResultString, err := dockercompose.Ps(workspaceFolder)
		if err != nil {
			return err
		}
		if dockerComposePsResultString == "" {
			fmt.Println("devcontainer already downed.")
			return nil
		}

		// 必要なのは最初の 1 行だけなので、最初の 1 行のみを取得
		dockerComposePsResultFirstItemString := strings.Split(dockerComposePsResultString, "\n")[0]

		// docker compose ps コマンドの結果からプロジェクト名を取得
		projectName, err := dockercompose.GetProjectName(dockerComposePsResultFirstItemString)
		if err != nil {
			return err
		}

		// プロジェクト名を使って docker compose stop を実行
		fmt.Printf("Run `docker compose -p %s stop`(Async)\n", projectName)

		// docker-compose.yaml の格納ディレクトリを探す
		dockerComposeFileDir, err := findDockerComposeFileDir()
		if err != nil {
			return err
		}

		// カレントディレクトリを記録して dockerComposeFileDir へ移動
		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}
		os.Chdir(dockerComposeFileDir)

		err = dockercompose.Stop(projectName)
		if err != nil {
			return err
		}

		// 元のカレントディレクトリへ戻る
		os.Chdir(currentDir)

	} else {
		// ワークスペースに対応するコンテナを探して ID を取得する
		containerID, err := docker.GetContainerIDFromWorkspaceFolder(workspaceFolder)
		if err != nil {
			return err
		}

		// 取得したコンテナに対して stop を行う
		fmt.Printf("Run `docker stop -f %s stop`(Async)\n", containerID)
		err = docker.Stop(containerID)
		if err != nil {
			return err
		}
	}
	return nil
}

func Down(args []string, devcontainerPath string, configDirForDevcontainer string) error {

	// `devcontainer read-configuration` で docker compose の利用判定

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]
	stdout, _ := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder)
	if stdout == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		return nil
	}

	// `dockerComposeFile` が含まれているかを確認する
	// 含まれているなら docker compose によるコンテナ構築がされている
	var configDir string
	if strings.Contains(stdout, "dockerComposeFile") {

		// docker-compose.yaml の格納ディレクトリを探す
		dockerComposeFileDir, err := findDockerComposeFileDir()
		if err != nil {
			return err
		}

		// カレントディレクトリを記録して dockerComposeFileDir へ移動
		currentDir, err := os.Getwd()
		if err != nil {
			return err
		}
		_, devcontainerJSONDir := findJSONInfo()

		os.Chdir(filepath.Join(devcontainerJSONDir, dockerComposeFileDir))

		// docker compose ps コマンドで compose の情報取得
		dockerComposePsResultString, err := dockercompose.Ps(workspaceFolder)
		if err != nil {
			return err
		}
		if dockerComposePsResultString == "" {
			fmt.Println("devcontainer already downed.")
			return nil
		}

		// 必要なのは最初の 1 行だけなので、最初の 1 行のみを取得
		dockerComposePsResultFirstItemString := strings.Split(dockerComposePsResultString, "\n")[0]

		// docker compose ps コマンドの結果からプロジェクト名を取得
		projectName, err := dockercompose.GetProjectName(dockerComposePsResultFirstItemString)
		if err != nil {
			return err
		}

		// プロジェクト名を使って docker compose down を実行
		fmt.Printf("Run `docker compose -p %s down`(Async)\n", projectName)
		err = dockercompose.Down(projectName)
		if err != nil {
			return err
		}

		// 元のカレントディレクトリへ戻る
		os.Chdir(currentDir)

		// pid ファイル参照のために、
		// コンテナ別の設定ファイル格納ディレクトリの名前(コンテナIDを記録)を記録
		configDir = util.GetConfigDir(configDirForDevcontainer, workspaceFolder)
	} else {
		// ワークスペースに対応するコンテナを探して ID を取得する
		containerID, err := docker.GetContainerIDFromWorkspaceFolder(workspaceFolder)
		if err != nil {
			return err
		}

		// 取得したコンテナに対して rm を行う
		fmt.Printf("Run `docker rm -f %s down`(Async)\n", containerID)
		err = docker.Rm(containerID)
		if err != nil {
			return err
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
		return err
	}
	pid, err := strconv.Atoi(string(pidStringBytes))
	if err != nil {
		return err
	}
	fmt.Printf("clipboard-data-receiver PID: %d\n", pid)
	tools.KillCdr(pid)

	err = os.RemoveAll(configDir)
	if err != nil {
		return err
	}
	return nil
}

// devcontainer.json の場所・ディレクトリを差がして返却する
func findJSONInfo() (string, string) {
	// devcontainer.json を取得
	var devcontainerJSONPath, devcontainerJSONDir string
	if util.IsExists(".devcontainer/devcontainer.json") {
		devcontainerJSONPath = ".devcontainer/devcontainer.json"
		devcontainerJSONDir = filepath.Dir(devcontainerJSONPath)
	} else if util.IsExists(".devcontainer.json") {
		devcontainerJSONPath = ".devcontainer.json"
		devcontainerJSONDir = filepath.Dir(devcontainerJSONPath)
	}

	return devcontainerJSONPath, devcontainerJSONDir
}

// docker-compose.yaml の格納ディレクトリを返却する
func findDockerComposeFileDir() (string, error) {
	// devcontainer.json を取得
	var devcontainerJSONPath, devcontainerJSONDir = findJSONInfo()

	// devcontainer.json 読み込み
	// fmt.Printf("devcontainerJSONPath directory: %s\n", devcontainerJSONPath)
	devcontainerJSONBytes, err := util.ParseJwcc(devcontainerJSONPath)
	if err != nil {
		return "", err
	}

	// docker-compose.yaml の格納ディレクトリを組み立て
	devcontainerJSON, err := UnmarshalDevcontainerJSON(devcontainerJSONBytes)
	if err != nil {
		return "", err
	}

	// string, []string を判別しながら docker-compose.yaml の場所を取得
	iDockerComposeFile := devcontainerJSON.DockerComposeFile
	var dockerComposeFilePath string
	switch v := iDockerComposeFile.(type) {
	case string:
		dockerComposeFilePath = v
	case []interface{}:
		vv := v[0].(string)
		dockerComposeFilePath = filepath.Join(devcontainerJSONDir, vv)
	default:
		return "", &UnknownTypeError{msg: "docker compose file path の型が不正です。 GitHub に issue を立てていただけるとありがたいです。"}
	}
	dockerComposeFileDir := filepath.Dir(dockerComposeFilePath)

	fmt.Printf("dockerComposeFileDir directory: %s\n", dockerComposeFileDir)
	return dockerComposeFileDir, nil
}

func GetConfigurationFilePath(devcontainerFilePath string, workspaceFolder string) (string, error) {
	stdout, _ := ReadConfiguration(devcontainerFilePath, "--workspace-folder", workspaceFolder)
	return GetConfigFilePath(stdout)
}

func ReadConfiguration(devcontainerFilePath string, readConfiguration ...string) (string, error) {
	args := append([]string{"read-configuration"}, readConfiguration...)
	result, err := Execute(devcontainerFilePath, args...)
	if err != nil {
		return "", errors.New("`devcontainer read-configuration` に失敗しました。`.devcontainer.json が存在することと、 docker エンジンが起動していることを確認してください。")
	}
	return result, err
}

func Templates(
	devcontainerFilePath string,
	workspaceFolder string,
	templateID string) (string, error) {
	// コマンドライン引数の末尾は `--workspace-folder` の値として使う

	args := []string{"templates", "apply", "--template-id", templateID, "--workspace-folder", workspaceFolder}
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
