package devcontainer

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

func buildDevcontainerStartVimExecArgs(containerID string, workspaceFolder string, shell string) []string {
	args := []string{
		"exec",
		"--container-id",
		containerID,
		"--workspace-folder",
		workspaceFolder,
	}

	if shell == "" {
		return append(args, "/VimRun.sh")
	}

	return append(args, shell)
}

// `devcontainer.vim start` 時の `devcontainer exec` の引数を組み立てる
//
// Args:
//   - containerID: コンテナ ID
//   - workspaceFolder: ワークスペースフォルダパス
//   - vimFileName: コンテナ上に転送した vim/nvim のファイル名
//   - useSystemVim: true の場合、システムにインストールした vim/nvim を利用する
//
// Return:
//
//	`devcontainer exec` に使うコマンドライン引数の配列
func devcontainerStartVimArgs(containerID string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configDirForDevcontainer string) ([]string, error) {
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
	vimLaunchScript := filepath.Join(configDirForDevcontainer, "VimRun.sh")
	if err := writeVimRunScript(vimLaunchScript, vimRunScript); err != nil {
		return nil, err
	}

	if err := docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh"); err != nil && shell == "" {
		return nil, fmt.Errorf("copy Vim launch script: %w", err)
	}

	return buildDevcontainerStartVimExecArgs(containerID, workspaceFolder, shell), nil
}
