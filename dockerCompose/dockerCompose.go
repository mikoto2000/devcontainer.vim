package dockerCompose

import (
	"os"
	"os/exec"
)

// `docker compose ps --format json` を実行し、結果の文字列を返却する。
func Ps(workspaceFolder string) (string, error) {

	// 現在のディレクトリを記憶
	currentDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 元のディレクトリへ戻る
	defer func() error {
		err := os.Chdir(currentDirectory)
		if err != nil {
			return err
		}
		return nil
	}()

	// ワークスペースまで移動
	err = os.Chdir(workspaceFolder)
	if err != nil {
		return "", err
	}

	dockerComposePsCommand := exec.Command("docker", "compose", "ps", "--format", "json")
	stdout, _ := dockerComposePsCommand.Output()
	return string(stdout), err
}

// `docker compose -p ${projectName} down` を実行する。
func Down(projectName string) error {
	dockerComposeDownCommand := exec.Command("docker", "compose", "-p", projectName, "down")
	return dockerComposeDownCommand.Start()
}
