package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

const APP_NAME = "devcontainer.vim"

type GetDirFunc func() (string, error)

func main() {
	// コマンドラインオプションのパース

	// Requirements のチェック
	// 1. docker
	isExistsDocker := isExistsCommand("docker")
	if !isExistsDocker {
		fmt.Fprintf(os.Stderr, "docker not found.")
		os.Exit(1)
	}

	// devcontainer.vim 用のディレクトリ作成
	// 1. ユーザーコンフィグ用ディレクトリ
	//    `os.UserConfigDir` + `devcontainer.vim`
	// 2. ユーザーキャッシュ用ディレクトリ
	//    `os.UserCacheDir` + `devcontainer.vim`
	createDirectory(os.UserConfigDir, APP_NAME)
	createDirectory(os.UserCacheDir, APP_NAME)

	// vim-appimage のダウンロード
	// 1. ユーザーキャッシュディレクトリ取得
	// 2. appimage がダウンロード済みかをチェックし、
	//    必要であればダウンロード

	// バックグラウンドでコンテナを起動
	// `docker run -d --rm os.Args[1:] sh -c "sleep infinity"`

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	// コンテナ停止
	// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`

}

func isExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		return false
	}
	return true
}

func createDirectory(pathFunc GetDirFunc, dirName string) {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appCacheDir = path.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, 0766); err != nil {
		panic(err)
	}
}
