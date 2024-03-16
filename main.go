package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
)

const APP_NAME = "devcontainer.vim"
const VIM_TAG_NAME = "v9.1.0181"
const VIM_DOWNLOAD_URL = "https://github.com/vim/vim-appimage/releases/download/%s/"
const VIM_FILE_NAME = "Vim-%s.glibc2.29-x86_64.AppImage"

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
	appCacheDir := createDirectory(os.UserCacheDir, APP_NAME)

	// vim-appimage のダウンロード
	// 1. ユーザーキャッシュディレクトリ取得
	// 2. appimage がダウンロード済みかをチェックし、
	//    必要であればダウンロード
	vimFileName := fmt.Sprintf(VIM_FILE_NAME, VIM_TAG_NAME)
	vimFilePath := path.Join(appCacheDir, vimFileName)

	if isExistsVimAppImage(vimFilePath) {
		fmt.Printf("Vim AppImage aleady exist, use %s.\n", vimFilePath)
	} else {
		vimDownloadUrl := fmt.Sprintf(VIM_DOWNLOAD_URL+vimFileName, VIM_TAG_NAME)
		fmt.Printf("Download Vim AppImage from %s ...", vimDownloadUrl)
		err := downloadVimAppImage(vimDownloadUrl, appCacheDir, vimFileName)
		if err != nil {
			panic(err)
		}
		fmt.Printf(" done.\n")
	}

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

func createDirectory(pathFunc GetDirFunc, dirName string) string {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appCacheDir = path.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, 0766); err != nil {
		panic(err)
	}
	return appCacheDir
}

func isExistsVimAppImage(vimFilePath string) bool {
	_, err := os.Stat(vimFilePath)
	return err == nil
}

func downloadVimAppImage(vimDownloadUrl string, appCacheDir string, vimFileName string) error {
	vimFilePath := path.Join(appCacheDir, vimFileName)

	// HTTP GETリクエストを送信
	resp, err := http.Get(vimDownloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ファイルを作成
	out, err := os.Create(vimFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// レスポンスの内容をファイルに書き込み
	_, err = io.Copy(out, resp.Body)
	return err
}
