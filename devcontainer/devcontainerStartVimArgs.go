package devcontainer

import (
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

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
func devcontainerStartVimArgs(containerID string, workspaceFolder string, vimFileName string, sendToTCP string, containerArch string, useSystemVim bool, shell string, configDirForDevcontainer string) []string {

	pattern := "pattern"
	var tmpl *template.Template
	var err error
	if useSystemVim {
		tmpl, err = template.New(pattern).Parse(vimRunX8664System)
		if err != nil {
			panic(err)
		}

	} else {
		// vim 起動シェルスクリプトを組み立てて転送
		if containerArch == "amd64" {
			if runtime.GOOS != "darwin" {
				tmpl, err = template.New(pattern).Parse(vimRunX8664AppImage)
				if err != nil {
					panic(err)
				}

			} else {
				tmpl, err = template.New(pattern).Parse(vimRunX8664Static)
				if err != nil {
					panic(err)
				}
			}
		} else {
			tmpl, err = template.New(pattern).Parse(vimRunAarch64)
			if err != nil {
				panic(err)
			}
		}
	}

	var vimRunScript strings.Builder
	tmplParams := map[string]string{
		"VimFileName": vimFileName,
		"SendToTcp":   sendToTCP,
	}
	err = tmpl.Execute(&vimRunScript, tmplParams)
	if err != nil {
		panic(err)
	}

	// Vim 起動スクリプトを出力
	vimLaunchScript := filepath.Join(configDirForDevcontainer, "VimRun.sh")
	os.RemoveAll(vimLaunchScript)
	err = os.WriteFile(vimLaunchScript, []byte(vimRunScript.String()), 0766)
	if err != nil {
		panic(err)
	}

	docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh")

	if shell == "" {
		return []string{
			"exec",
			"--container-id",
			containerID,
			"--workspace-folder",
			workspaceFolder,
			"/VimRun.sh",
		}
	} else {
		return []string{
			"exec",
			"--container-id",
			containerID,
			"--workspace-folder",
			workspaceFolder,
			shell,
		}
	}
}
