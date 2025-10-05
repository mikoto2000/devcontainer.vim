package dockercompose

import (
	"os"
	"os/exec"
)

const containerCommand = "docker"

type PsCommandError struct {
	msg string
}

func (e *PsCommandError) Error() string {
	return e.msg
}

type StopCommandError struct {
	msg string
}

func (e *StopCommandError) Error() string {
	return e.msg
}

type DownCommandError struct {
	msg string
}

func (e *DownCommandError) Error() string {
	return e.msg
}

// `docker compose ps --format json` を実行し、結果の文字列を返却する。
func Ps(workspaceFolder string) (string, error) {

	// 現在のディレクトリを記憶
	currentDirectory, err := os.Getwd()
	if err != nil {
		return "", &PsCommandError{msg: "Failed to get current directory"}
	}

	// 元のディレクトリへ戻る
	defer func() error {
		err := os.Chdir(currentDirectory)
		if err != nil {
			return &PsCommandError{msg: "Failed to change directory"}
		}
		return nil
	}()

	// ワークスペースまで移動
	err = os.Chdir(workspaceFolder)
	if err != nil {
		return "", &PsCommandError{msg: "ワークスペースへの移動に失敗しました。指定したディレクトリが存在するか・パーミッションが正しいかの確認をしてください。 "}
	}

	dockerComposePsCommand := exec.Command(containerCommand, "compose", "ps", "--all", "--format", "json")
	stdout, err := dockerComposePsCommand.Output()
	if err != nil {
		return "", &PsCommandError{msg: "docker compose ps コマンドの実行に失敗しました。docker がインストールされているか・docker エンジンが起動しているかの確認をしてください。 "}
	}
	return string(stdout), err
}

// `docker compose -p ${projectName} stop` を実行する。
func Stop(projectName string) error {
	dockerComposeStopCommand := exec.Command(containerCommand, "compose", "-p", projectName, "stop")
	err := dockerComposeStopCommand.Start()
	if err != nil {
		return &StopCommandError{msg: "docker compose stop コマンドの実行に失敗しました。docker がインストールされているか・docker エンジンが起動しているかの確認をしてください。 "}
	}
	return nil
}

// `docker compose -p ${projectName} down` を実行する。
func Down(projectName string) error {
	dockerComposeDownCommand := exec.Command(containerCommand, "compose", "-p", projectName, "down")
	err := dockerComposeDownCommand.Start()
	if err != nil {
		return &DownCommandError{msg: "docker compose down コマンドの実行に失敗しました。docker がインストールされているか・docker エンジンが起動しているかの確認をしてください。 "}
	}
	return nil
}
