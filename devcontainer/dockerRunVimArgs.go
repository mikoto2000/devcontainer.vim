package devcontainer

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

func buildDockerRunVimExecArgs(containerID string, shell string) []string {
	if shell == "" {
		return []string{
			"exec",
			"-it",
			containerID,
			"/VimRun.sh",
		}
	}

	return []string{
		"exec",
		"-it",
		containerID,
		shell,
	}
}

// `devcontainer.vim run` 時の `docker exec` の引数を組み立てる
//
// Args:
//   - containerID: コンテナ ID
//   - vimFileName: コンテナ上に転送した vim のファイル名
//   - useSystemVim: true の場合、システムにインストールされた vim/nvim を使用する
//
// Return:
//
//	`docker exec` に使うコマンドライン引数の配列
func dockerRunVimArgs(containerID string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, shell string, configFilePath string) ([]string, error) {
	var templateSource string
	var err error
	if useSystemVim {
		templateSource = vimRunX8664System
	} else {
		if containerArch == "amd64" {
			if runtime.GOOS != "darwin" {
				templateSource = vimRunX8664AppImage
			} else {
				templateSource = vimRunX8664Static
			}
		} else {
			templateSource = vimRunAarch64
		}
	}

	tmuxCommand := "/" + tmuxFileName
	if useSystemTmux {
		tmuxCommand = tmuxFileName
	}
	vimRunScript, err := renderVimRunScript(templateSource, vimRunScriptParams{
		VimFileName: vimFileName,
		SendToTcp:   sendToTCP,
		TmuxCommand: tmuxCommand,
	})
	if err != nil {
		return nil, err
	}

	// Vim 起動スクリプトを出力
	vimLaunchScript := filepath.Join(configFilePath, "VimRun.sh")
	os.RemoveAll(vimLaunchScript)
	err = os.WriteFile(vimLaunchScript, []byte(vimRunScript), 0766)
	if err != nil {
		return nil, err
	}

	docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh")

	return buildDockerRunVimExecArgs(containerID, shell), nil
}
