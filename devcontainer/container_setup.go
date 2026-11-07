package devcontainer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// コンテナのアーキテクチャを取得する
func getContainerArch(containerID string) (string, error) {
	containerArch, err := docker.Exec(containerID, "uname", "-m")
	if err != nil {
		return "", err
	}
	containerArch = strings.TrimSpace(containerArch)
	containerArch, err = util.NormalizeContainerArch(containerArch)
	if err != nil {
		return "", err
	}
	fmt.Printf("Container Arch: '%s'.\n", containerArch)
	return containerArch, nil
}

// port-forwarderをコンテナにインストールする
func installPortForwarder(containerID, vimInstallDir, containerArch string) error {
	portForwarderContainerPath, err := tools.PortForwarderContainer(tools.DefaultInstallerUseServices{}).Install(vimInstallDir, containerArch, false)
	if err != nil {
		return err
	}
	err = docker.Cp("port-forwarder-container", portForwarderContainerPath, containerID, "/port-forwarder")
	if err != nil {
		return err
	}
	return nil
}

// tmuxの検出とインストールを行う
func setupTmux(containerID, vimInstallDir string, containerArch string) (string, bool, error) {
	useSystemTmux := false
	fmt.Printf("Check system installed tmux ... ")
	out, _ := docker.Exec(containerID, "which", "tmux")
	if out != "" {
		fmt.Printf("found.\n")
		useSystemTmux = true
	} else {
		fmt.Printf("not found.\n")
	}
	fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))

	if useSystemTmux {
		return "tmux", true, nil
	}

	tmuxFilePath, err := tools.InstallTmux(vimInstallDir, containerArch)
	if err != nil {
		return "", false, err
	}

	err = docker.Cp("tmux", tmuxFilePath, containerID, "/tmux")
	if err != nil {
		return "", false, err
	}

	dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /tmux"}
	fmt.Printf("Chown tmux: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		return "", false, &ChmodError{msg: "chmod error."}
	}
	fmt.Printf(" done.\n")

	return "tmux", false, nil
}

// Vimの検出とインストールを行う
func setupVim(containerID, vimInstallDir string, nvim bool, containerArch string) (string, bool, error) {
	vimFileName := "vim"
	if nvim {
		vimFileName = "nvim"
	}

	fmt.Printf("Check system installed %s ... ", vimFileName)
	out, _ := docker.Exec(containerID, "which", vimFileName)
	if out != "" {
		fmt.Printf("found.\n")
		fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))
		return vimFileName, true, nil
	}
	fmt.Printf("not found.\n")
	fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))

	vimFilePath, err := tools.InstallVim(vimInstallDir, nvim, containerArch)
	if err != nil {
		return "", false, err
	}

	// The cached artifact is named vim_<arch> or nvim_<arch>. Use the
	// installed editor for both the launch command and clipboard script.
	vimFileName, _, _ = strings.Cut(filepath.Base(vimFilePath), "_")
	if err := docker.Cp("vim", vimFilePath, containerID, "/"+vimFileName); err != nil {
		return vimFileName, false, err
	}

	dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		return vimFileName, false, &ChmodError{msg: "chmod error."}
	}
	fmt.Printf(" done.\n")

	return vimFileName, false, nil
}

// Vimファイル（SendToTcp.vimとvimrc）をコンテナに転送する
func transferVimFiles(containerID, configDir, vimrc string, noCdr bool, port int, isNvim bool) (string, error) {
	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDir, port, noCdr, isNvim)
	if err != nil {
		return "", err
	}

	// コンテナへ SendToTcp.vim を転送
	err = docker.Cp("SendToTcp.vim", sendToTCP, containerID, "/")
	if err != nil {
		return sendToTCP, err
	}

	// コンテナへ vimrc を転送
	err = docker.Cp("vimrc", vimrc, containerID, "/")
	if err != nil {
		return sendToTCP, err
	}

	return sendToTCP, nil
}
