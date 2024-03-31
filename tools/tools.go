package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// ツール情報
type Tool struct {
	FileName    string
	DownloadUrl string
	installFunc func(downloadUrl string, installDir string, name string) (string, error)
}

// ツールのインストールを実行
func (t Tool) Install(installDir string) (string, error) {
	return t.installFunc(t.DownloadUrl, installDir, t.FileName)
}

// 単純なファイル配置でインストールが完了するもののインストール処理。
//
// downloadUrl からファイルをダウンロードし、 installDir に fileName とう名前で配置する。
func simpleInstall(downloadUrl string, installDir string, fileName string) (string, error) {

	// ツールの配置先組み立て
	filePath := filepath.Join(installDir, fileName)

	// ツールのダウンロード
	err := download(downloadUrl, filePath)
	if err != nil {
		return filePath, err
	}

	// 実行権限の付与
	err = util.AddExecutePermission(filePath)
	if err != nil {
		return filePath, err
	}

	return filePath, nil
}

// Vim のダウンロード URL
const VIM_DOWNLOAD_URL = "https://github.com/vim/vim-appimage/releases/download/v9.1.0181/Vim-v9.1.0181.glibc2.29-x86_64.AppImage"

// Vim のツール情報
var VIM Tool = Tool{
	FileName:    "vim",
	DownloadUrl: VIM_DOWNLOAD_URL,
	installFunc: func(downloadUrl string, installDir string, fileName string) (string, error) {
		return simpleInstall(downloadUrl, installDir, fileName)
	},
}

// devcontainer/cli のツール情報
var DEVCONTAINER Tool = Tool{
	FileName:    DEVCONTAINER_FILE_NAME,
	DownloadUrl: DOWNLOAD_URL_DEVCONTAINERS_CLI,
	installFunc: func(downloadUrl string, installDir string, fileName string) (string, error) {
		return simpleInstall(downloadUrl, installDir, fileName)
	},
}

// ファイルダウンロード処理。
//
// downloadUrl からファイルをダウンロードし、 destPath へ配置する。
func download(downloadUrl string, destPath string) error {
	if util.IsExists(destPath) {
		fmt.Printf("%s aleady exist, use this.\n", filepath.Base(destPath))
	} else {
		fmt.Printf("Download %s from %s ...", filepath.Base(destPath), downloadUrl)

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

		fmt.Printf(" done.\n")
	}

	return nil
}
