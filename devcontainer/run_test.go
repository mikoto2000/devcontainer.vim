package devcontainer

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func TestSetupContainer(t *testing.T) {
	appName := "devcontainer.vim"

	_, binDir, configDirForDocker, _, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		panic(err)
	}

	nvim := false
	cdrPath, err := tools.InstallRunTools(binDir, nvim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing run tools: %v\n", err)
		os.Exit(1)
	}

	vimrc := "../test/resource/TestRun/vimrc"

	containerID, _, _, _, _, _, _, err := setupContainer(
		[]string{"mcr.microsoft.com/devcontainers/base:bookworm"},
		cdrPath,
		binDir,
		nvim,
		configDirForDocker,
		vimrc,
		[]string{},
	)

	if err != nil {
		t.Fatalf("error: %s", err)
	}

	// 後片付け
	// コンテナ停止
	defer func() {
		// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
		fmt.Printf("Stop container(Async) %s.\n", containerID)
		err = exec.Command(containerCommand, "stop", containerID).Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}
	}()

	//     /vim
	vimOutput, err := docker.Exec(containerID, "sh", "-c", "ls /vim*")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
	}
	//     /vimrc
	vimrcOutput, err := docker.Exec(containerID, "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
}
