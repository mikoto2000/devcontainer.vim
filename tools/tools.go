package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mikoto2000/devcontainer.vim/util"
)

type Tools struct {
	Vim string
}

type Tool struct {
	Name    string
	Install func(installDir string) (string, error)
}

func InstallTools(installDir string) (Tools, error) {
	vim, err := VIM.Install(installDir)
	if err != nil {
		panic(err)
	}

	return Tools{
		Vim: vim,
	}, nil
}

const APP_NAME = "devcontainer.vim"

const VIM_TAG_NAME = "v9.1.0181"
const VIM_DOWNLOAD_URL = "https://github.com/vim/vim-appimage/releases/download/%s/"
const VIM_FILE_NAME = "Vim-%s.glibc2.29-x86_64.AppImage"

var VIM Tool = Tool{
	Name: "vim",
	Install: func(installDir string) (string, error) {

		// Vim 関連の文字列組み立て
		vimFileName := fmt.Sprintf(VIM_FILE_NAME, VIM_TAG_NAME)
		vimFilePath := filepath.Join(installDir, vimFileName)

		// vim-appimage のダウンロード
		// 1. ユーザーキャッシュディレクトリ取得
		// 2. appimage がダウンロード済みかをチェックし、
		//    必要であればダウンロード
		if util.IsExists(vimFilePath) {
			fmt.Printf("Vim AppImage aleady exist, use %s.\n", vimFilePath)
		} else {
			vimDownloadUrl := fmt.Sprintf(VIM_DOWNLOAD_URL+vimFileName, VIM_TAG_NAME)
			fmt.Printf("Download Vim AppImage from %s ...", vimDownloadUrl)
			err := download(vimDownloadUrl, vimFilePath)
			if err != nil {
				return vimFilePath, err
			}
			fmt.Printf(" done.\n")
		}
		return vimFilePath, nil
	},
}

func download(downloadUrl string, destPath string) error {

	// HTTP GETリクエストを送信
	resp, err := http.Get(downloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ファイルを作成
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// レスポンスの内容をファイルに書き込み
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
