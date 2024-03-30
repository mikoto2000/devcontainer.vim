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
		vimDownloadUrl := fmt.Sprintf(VIM_DOWNLOAD_URL+vimFileName, VIM_TAG_NAME)
		err := download(vimDownloadUrl, vimFilePath)
		if err != nil {
			return vimFilePath, err
		}
		return vimFilePath, nil
	},
}

var DEVCONTAINER Tool = Tool{
	Name: "devcontainer",
	Install: func(installDir string) (string, error) {
		devcontainerFileName := filepath.Base(DOWNLOAD_URL_DEVCONTAINERS_CLI)
		devcontainerFilePath := filepath.Join(installDir, devcontainerFileName)

		// devcontainers-cli のダウンロード
		err := download(DOWNLOAD_URL_DEVCONTAINERS_CLI, devcontainerFilePath)
		if err != nil {
			panic(err)
		}

		return devcontainerFilePath, nil
	},
}

func download(downloadUrl string, destPath string) error {
	if util.IsExists(destPath) {
		fmt.Printf("%s aleady exist, use this.\n", filepath.Base(destPath))
	} else {
		fmt.Printf("Download %s from %s ...", filepath.Base(downloadUrl), downloadUrl)

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
