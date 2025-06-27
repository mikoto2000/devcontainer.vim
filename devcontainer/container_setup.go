package devcontainer

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// Vimの検出とインストールを行う
func setupVim(containerID, vimInstallDir string, nvim bool, containerArch string) (string, bool, error) {
	vimFileName := "vim"
	if nvim {
		vimFileName = "nvim"
	}

	useSystemVim := false
	fmt.Printf("Check system installed %s ... ", vimFileName)
	out, _ := docker.Exec(containerID, "which", vimFileName)
	if out != "" {
		fmt.Printf("found.\n")
		useSystemVim = true

		if nvim {
			vimFileName = "nvim"
		}
	} else {
		fmt.Printf("not found.\n")

		if runtime.GOARCH == "arm64" {
			// arm の場合スタティックリンクの nvim を作れないため、 vim にフォールバック
			vimFileName = "vim"
			nvim = false
		} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" && nvim {
			// M1 Mac で amd64 のコンテナを動かすと、なぜか AppImage が動かないので vim にフォールバック
			vimFileName = "vim"
			nvim = false
		}
	}
	fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))

	if !useSystemVim {
		// コンテナへ Vim/Neovim を転送して実行権限を追加
		vimFilePath, err := tools.InstallVim(vimInstallDir, nvim, containerArch)
		if err != nil {
			return "", false, err
		}

		// start.goとrun.goで異なる処理: start.goは特別なパス解析が必要
		actualVimFileName := vimFileName
		if strings.Contains(vimFilePath, "_") {
			// vim_<ARCH>, nvim_<ARCH> の形式でパスがわたってくる場合（start.go）
			actualVimFileName = strings.Split(filepath.Base(vimFilePath), "_")[0]
		}

		err = docker.Cp("vim", vimFilePath, containerID, "/"+actualVimFileName)
		if err != nil {
			return actualVimFileName, useSystemVim, err
		}

		// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
		dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + actualVimFileName}
		fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
		chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
		if err != nil {
			fmt.Fprintln(os.Stderr, "chmod error.")
			fmt.Fprintln(os.Stderr, string(chmodResult))
			return actualVimFileName, useSystemVim, &ChmodError{msg: "chmod error."}
		}
		fmt.Printf(" done.\n")
		
		return actualVimFileName, useSystemVim, nil
	}

	return vimFileName, useSystemVim, nil
}

// Vimファイル（SendToTcp.vimとvimrc）をコンテナに転送する
func transferVimFiles(containerID, configDir, vimrc string, port int, isNvim bool) (string, error) {
	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDir, port, isNvim)
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