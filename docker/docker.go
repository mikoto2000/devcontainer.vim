package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const containerCommand = "docker"

type ContainerNotFoundError struct {
	msg string
}

func (e *ContainerNotFoundError) Error() string {
	return e.msg
}

// workspaceFolder で指定したディレクトリに対応するコンテナのコンテナ ID を返却する
func GetContainerIDFromWorkspaceFolder(workspaceFolder string) (string, error) {

	// `devcontainer.local_folder=${workspaceFolder}` が含まれている行を探す

	workspaceFilderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return "", err
	}

	psResult, err := Ps("label=devcontainer.local_folder=" + workspaceFilderAbs)
	if psResult == "" {
		return "", &ContainerNotFoundError{msg: "container not found."}
	}
	if err != nil {
		return "", err
	}

	id, err := GetID(psResult)
	if err != nil {
		return "", err
	}

	return id, nil
}

// `docker exec` コマンドを実行する。
func Exec(containerID string, command ...string) (string, error) {

	dockerExecArgs := []string{"exec", "-t", containerID}
	dockerExecArgs = append(dockerExecArgs, command...)

	dockerExecCommand := exec.Command(containerCommand, dockerExecArgs...)
	stdout, err := dockerExecCommand.Output()
	return string(stdout), err
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
