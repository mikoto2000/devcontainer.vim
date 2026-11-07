package devcontainer

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// Both probe results exit successfully so Docker execution errors stay distinct.
const containerToolProbe = `if command -v "$1" >/dev/null; then printf 'found\n'; else printf 'missing\n'; fi`

func probeContainerTool(containerID, name string) (bool, error) {
	out, err := docker.Exec(containerID, "sh", "-c", containerToolProbe, "devcontainer-vim-tool-probe", name)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			out += "\n" + string(exit.Stderr)
		}
		if diagnostic := strings.TrimSpace(out); diagnostic != "" {
			return false, fmt.Errorf("detect %s in container %s: %w: %s", name, containerID, err, diagnostic)
		}
		return false, fmt.Errorf("detect %s in container %s: %w", name, containerID, err)
	}
	switch strings.TrimSpace(out) {
	case "found":
		fmt.Printf("System %s found.\n", name)
		return true, nil
	case "missing":
		fmt.Printf("System %s not found.\n", name)
		return false, nil
	default:
		return false, fmt.Errorf("detect %s in container %s: unexpected probe output %q", name, containerID, out)
	}
}

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
	const containerPath = "/port-forwarder"
	if err := docker.Cp("port-forwarder-container", portForwarderContainerPath, containerID, containerPath); err != nil {
		return err
	}
	// The copy can carry permissions from a non-executable host cache entry.
	cmd := exec.Command(containerCommand, "exec", "--user", "root", containerID, "chmod", "a+x", containerPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("make %s executable in container %s: %w: %s", containerPath, containerID, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// tmuxの検出とインストールを行う
func setupTmux(containerID, vimInstallDir string, containerArch string) (string, bool, error) {
	useSystemTmux, err := probeContainerTool(containerID, "tmux")
	if err != nil {
		return "", false, err
	}
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

	dockerChmodArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /tmux"}
	fmt.Printf("Make tmux executable: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChmodArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChmodArgs...).CombinedOutput()
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

	useSystemVim, err := probeContainerTool(containerID, vimFileName)
	if err != nil {
		return "", false, err
	}
	if useSystemVim {
		return vimFileName, true, nil
	}

	vimFilePath, vimFileName, err := tools.InstallContainerEditor(vimInstallDir, nvim, containerArch)
	if err != nil {
		return "", false, err
	}

	if err := docker.Cp("vim", vimFilePath, containerID, "/"+vimFileName); err != nil {
		return vimFileName, false, err
	}

	dockerChmodArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Make %s executable: `%s \"%s\"` ...", vimFileName, containerCommand, strings.Join(dockerChmodArgs, "\" \""))
	chmodResult, err := exec.Command(containerCommand, dockerChmodArgs...).CombinedOutput()
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
