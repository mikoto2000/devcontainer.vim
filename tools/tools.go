package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// ツール情報
type Tool struct {
	FileName             string
	CalculateDownloadUrl func() string
	installFunc          func(downloadUrl string, installDir string, name string, override bool) (string, error)
}

// ツールのインストールを実行
func (t Tool) Install(installDir string, override bool) (string, error) {
	return t.installFunc(t.CalculateDownloadUrl(), installDir, t.FileName, override)
}

// 単純なファイル配置でインストールが完了するもののインストール処理。
//
// downloadUrl からファイルをダウンロードし、 installDir に fileName とう名前で配置する。
func simpleInstall(downloadUrl string, installDir string, fileName string, override bool) (string, error) {

	// ツールの配置先組み立て
	filePath := filepath.Join(installDir, fileName)

	// ツールのダウンロード
	err := download(downloadUrl, filePath, override)
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
const VIM_DOWNLOAD_URL_PATTERN = "https://github.com/vim/vim-appimage/releases/download/{{ .TagName }}/Vim-{{ .TagName }}.glibc2.29-x86_64.AppImage"

// Vim のツール情報
var VIM Tool = Tool{
	FileName: "vim",
	CalculateDownloadUrl: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("vim", "vim-appimage")
		if err != nil {
			panic(err)
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(VIM_DOWNLOAD_URL_PATTERN)
		if err != nil {
			panic(err)
		}

		tmplParams := map[string]string{"TagName": latestTagName}
		var downloadUrl strings.Builder
		err = tmpl.Execute(&downloadUrl, tmplParams)
		if err != nil {
			panic(err)
		}
		return downloadUrl.String()
	},
	installFunc: func(downloadUrl string, installDir string, fileName string, override bool) (string, error) {
		return simpleInstall(downloadUrl, installDir, fileName, override)
	},
}

// ファイルダウンロード処理。
//
// downloadUrl からファイルをダウンロードし、 destPath へ配置する。
func download(downloadUrl string, destPath string, override bool) error {
	if util.IsExists(destPath) && !override {
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
