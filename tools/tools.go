package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/mikoto2000/devcontainer.vim/util"
)

// ツール情報
type Tool struct {
	FileName             string
	CalculateDownloadURL func() string
	installFunc          func(downloadURL string, filePath string) (string, error)
}

// ツールのインストールを実行
func (t Tool) Install(installDir string, override bool) (string, error) {

	// ツールの配置先組み立て
	filePath := filepath.Join(installDir, t.FileName)

	if util.IsExists(filePath) && !override {
		fmt.Printf("%s aleady exist, use this.\n", filePath)
		return filePath, nil
	} else {
		return t.installFunc(t.CalculateDownloadURL(), filePath)
	}
}

// 単純なファイル配置でインストールが完了するもののインストール処理。
//
// downloadURL からファイルをダウンロードし、 installDir に fileName とう名前で配置する。
func simpleInstall(downloadURL string, filePath string) (string, error) {

	// ツールのダウンロード
	err := download(downloadURL, filePath)
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
const vimDownloadURLPattern = "https://github.com/vim/vim-appimage/releases/download/{{ .TagName }}/Vim-{{ .TagName }}.glibc2.29-x86_64.AppImage"

// Vim のツール情報
var VIM Tool = Tool{
	FileName: "vim",
	CalculateDownloadURL: func() string {
		latestTagName, err := util.GetLatestReleaseFromGitHub("vim", "vim-appimage")
		if err != nil {
			panic(err)
		}

		pattern := "pattern"
		tmpl, err := template.New(pattern).Parse(vimDownloadURLPattern)
		if err != nil {
			panic(err)
		}

		tmplParams := map[string]string{"TagName": latestTagName}
		var downloadURL strings.Builder
		err = tmpl.Execute(&downloadURL, tmplParams)
		if err != nil {
			panic(err)
		}
		return downloadURL.String()
	},
	installFunc: func(downloadURL string, filePath string) (string, error) {
		return simpleInstall(downloadURL, filePath)
	},
}

// 進捗表示用構造体
type ProgressWriter struct {
	Total   int64
	Current int64
}

func (p *ProgressWriter) Write(data []byte) (int, error) {
	n := len(data)
	p.Current += int64(n)

	percentage := float64(p.Current) / float64(p.Total) * 100.0
	fmt.Printf("%6.2f%%", percentage)

	// カーソルを 7 文字戻す
	fmt.Printf("\033[7D")

	return n, nil
}

// ファイルダウンロード処理。
//
// downloadURL からファイルをダウンロードし、 destPath へ配置する。
func download(downloadURL string, destPath string) error {
	fmt.Printf("Download %s from %s ...", filepath.Base(destPath), downloadURL)

	// HTTP GETリクエストを送信
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size := resp.ContentLength

	// ファイルを作成
	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	progress := &ProgressWriter{
		Total: size,
	}

	// レスポンスの内容をファイルに書き込み
	_, err = io.Copy(out, io.TeeReader(resp.Body, progress))
	if err != nil {
		return err
	}

	fmt.Printf(" done. \n")

	return nil
}

// run サブコマンド用のツールインストール
func InstallRunTools(installDir string) (string, string, error) {
	vimPath, err := VIM.Install(installDir, false)
	if err != nil {
		return vimPath, "", err
	}
	cdrPath, err := CDR.Install(installDir, false)
	if err != nil {
		return vimPath, cdrPath, err
	}
	return vimPath, cdrPath, err
}

// start サブコマンド用のツールインストール
// 戻り値は、 vimPath, devcontainerPath, cdrPath, error
func InstallStartTools(installDir string) (string, string, string, error) {
	vimPath, err := VIM.Install(installDir, false)
	if err != nil {
		return vimPath, "", "", err
	}
	devcontainerPath, err := DEVCONTAINER.Install(installDir, false)
	if err != nil {
		return vimPath, devcontainerPath, "", err
	}
	cdrPath, err := CDR.Install(installDir, false)
	if err != nil {
		return vimPath, devcontainerPath, cdrPath, err
	}
	return vimPath, devcontainerPath, cdrPath, err
}

// devcontainer サブコマンド用のツールインストール
func InstallDevcontainerTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, false)
	return devcontainerPath, err
}

// Templates サブコマンド用のツールインストール
func InstallTemplatesTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, false)
	return devcontainerPath, err
}

// Stop サブコマンド用のツールインストール
func InstallStopTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, false)
	return devcontainerPath, err
}

// Down サブコマンド用のツールインストール
func InstallDownTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, false)
	return devcontainerPath, err
}

// SelfUpdate downloads the latest release of devcontainer.vim from GitHub and replaces the current binary
func SelfUpdate() error {
	// Get the latest release tag name from GitHub
	latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "devcontainer.vim")
	if err != nil {
		return err
	}

	// Construct the download URL for the latest release
	var downloadURL string
	if runtime.GOOS == "windows" {
		downloadURL = fmt.Sprintf("https://github.com/mikoto2000/devcontainer.vim/releases/download/%s/devcontainer.vim-windows-amd64.exe", latestTagName)
	} else if runtime.GOOS == "darwin" {
		downloadURL = fmt.Sprintf("https://github.com/mikoto2000/devcontainer.vim/releases/download/%s/devcontainer.vim-darwin-amd64", latestTagName)
	} else {
		downloadURL = fmt.Sprintf("https://github.com/mikoto2000/devcontainer.vim/releases/download/%s/devcontainer.vim-linux-amd64", latestTagName)
	}

	// Download the latest release
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	// Rename the current binary to avoid "text file busy" error
	tempPath := executablePath + ".old"
	err = os.Rename(executablePath, tempPath)
	if err != nil {
		return err
	}

	_, err = simpleInstall(downloadURL, executablePath)
	if err != nil {
		// Restore the original binary if download fails
		os.Rename(tempPath, executablePath)
		return err
	}

	// Remove the old binary
	os.Remove(tempPath)

	fmt.Println("devcontainer.vim has been updated to the latest version.")
	return nil
}
