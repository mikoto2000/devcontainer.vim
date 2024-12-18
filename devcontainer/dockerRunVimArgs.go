package devcontainer

import (
	"html/template"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

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
func dockerRunVimArgs(containerID string, vimFileName string, sendToTCP string, containerArch string, useSystemVim bool, shell string, configFilePath string) []string {

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
	vimLaunchScript := filepath.Join(configFilePath, "VimRun.sh")
	os.RemoveAll(vimLaunchScript)
	err = os.WriteFile(vimLaunchScript, []byte(vimRunScript.String()), 0766)
	if err != nil {
		panic(err)
	}

	docker.Cp("Vim launch script", vimLaunchScript, containerID, "/VimRun.sh")

	if shell == "" {
		return []string{
			"exec",
			"-it",
			containerID,
			"/VimRun.sh",
		}
	} else {
		return []string{
			"exec",
			"-it",
			containerID,
			shell,
		}
	}
}
