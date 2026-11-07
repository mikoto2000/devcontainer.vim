package devcontainer

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

const dockerDetachKeys = "ctrl-\\\\"

func buildDockerRunVimExecArgs(containerID string, shell string) []string {
	if shell == "" {
		return []string{
			"exec",
			"--detach-keys",
			dockerDetachKeys,
			"-it",
			containerID,
			"/VimRun.sh",
		}
	}

	return []string{
		"exec",
		"--detach-keys",
		dockerDetachKeys,
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
func dockerRunVimArgs(containerID string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePath string) ([]string, error) {
	templateSource := selectVimRunTemplate(runtime.GOOS, containerArch, useSystemVim)

	tmuxCommand := "/" + tmuxFileName
	if useSystemTmux {
		tmuxCommand = tmuxFileName
	}
	vimRunScript, err := renderVimRunScript(templateSource, vimRunScriptParams{
		VimFileName: vimFileName,
		SendToTcp:   sendToTCP,
		UseTmux:     !noTmux,
		TmuxCommand: tmuxCommand,
	})
	if err != nil {
		return nil, err
	}

	// Vim 起動スクリプトを出力
	vimLaunchScript := filepath.Join(configFilePath, "VimRun.sh")
	if err := writeVimRunScript(vimLaunchScript, vimRunScript); err != nil {
		return nil, err
	}

	if err := docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh"); err != nil && shell == "" {
		return nil, fmt.Errorf("copy Vim launch script: %w", err)
	}

	return buildDockerRunVimExecArgs(containerID, shell), nil
}
