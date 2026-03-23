package devcontainer

import (
	"os"
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
func devcontainerStartVimArgs(containerID string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, shell string, configDirForDevcontainer string) ([]string, error) {
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
	vimLaunchScript := filepath.Join(configDirForDevcontainer, "VimRun.sh")
	os.RemoveAll(vimLaunchScript)
	err = os.WriteFile(vimLaunchScript, []byte(vimRunScript), 0766)
	if err != nil {
		return nil, err
	}

	docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh")

	return buildDevcontainerStartVimExecArgs(containerID, workspaceFolder, shell), nil
}
