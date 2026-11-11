package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Command is the executable name of the Docker CLI.
const Command = "docker"

type ContainerNotFoundError struct {
	msg string
}

func (e *ContainerNotFoundError) Error() string {
	return e.msg
}

// workspaceFolder で指定したディレクトリに対応するコンテナのコンテナ ID を返却する
func GetContainerIDFromWorkspaceFolder(workspaceFolder string) (string, error) {

	// `devcontainer.local_folder=${workspaceFolder}` が含まれている行を探す

	workspaceFolderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return "", err
	}

	psResult, err := ps("label=devcontainer.local_folder="+workspaceFolderAbs, true)
	if err != nil {
		return "", err
	}
	return GetID(psResult)
}

// Exec runs a noninteractive command and returns its standard output.
func Exec(containerID string, command ...string) (string, error) {

	dockerExecArgs := []string{"exec", containerID}
	dockerExecArgs = append(dockerExecArgs, command...)

	dockerExecCommand := exec.Command(Command, dockerExecArgs...)
	stdout, err := dockerExecCommand.Output()
	return string(stdout), err
}

// `docker ps --format json` コマンドを実行する。
func Ps(filter string) (string, error) {
	return ps(filter, false)
}

func ps(filter string, includeStopped bool) (string, error) {
	args := []string{"ps", "--format", "json"}
	if filter != "" {
		args = append(args, "--filter", filter)
	}
	if includeStopped {
		args = append(args, "--all")
	}
	dockerPsCommand := exec.Command(Command, args...)
	stdout, err := dockerPsCommand.Output()
	return string(stdout), err
}

// `docker stop ${containerID}` を実行し、完了を待つ。
func Stop(containerID string) error {
	return runContainerCommand("stop", containerID)
}

// `docker rm -f ${containerID}` コマンドを実行する。
func Rm(containerID string) error {
	return runContainerCommand("rm", "-f", containerID)
}

func runContainerCommand(args ...string) error {
	output, err := exec.Command(Command, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}

func Cp(tagForLog string, from string, containerID string, to string) error {
	dockerCpArgs := []string{"cp", from, containerID + ":" + to}
	fmt.Printf("Copy %s: `%s \"%s\"` ...", tagForLog, Command, strings.Join(dockerCpArgs, "\" \""))
	copyResult, err := exec.Command(Command, dockerCpArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "copy error.")
		fmt.Fprintln(os.Stderr, string(copyResult))
		return err
	}
	fmt.Printf(" done.\n")
	return nil
}
