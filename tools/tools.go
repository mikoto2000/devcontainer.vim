package tools

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// ツール情報
type Tool struct {
	FileName             string
	CalculateDownloadURL func(containerArch string) (string, error)
	installFunc          func(downloadURL string, filePath string, containerArch string) (string, error)
}

// ツールのインストールを実行
func (t Tool) Install(installDir string, containerArch string, override bool) (string, error) {

	// tool download から直接呼ばれることもあるのでここでも正規化する
	containerArch, err := util.NormalizeContainerArch(containerArch)
	if err != nil {
		return "", nil
	}

	// ツールの配置先組み立て
	var fileName string
	if containerArch != "" {
		fileName = t.FileName + "_" + containerArch
	} else {
		fileName = t.FileName
	}
	filePath := filepath.Join(installDir, fileName)

	if util.IsExists(filePath) && !override {
		fmt.Printf("%s aleady exist, use this.\n", filePath)
		return filePath, nil
	} else {
		downloadURL, err := t.CalculateDownloadURL(containerArch)
		if err != nil {
			return "", err
		}
		return t.installFunc(downloadURL, filePath, containerArch)
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
func InstallRunTools(installDir string, nvim bool) (string, error) {
	var err error
	cdrPath, err := CDR.Install(installDir, "", false)
	if err != nil {
		return cdrPath, err
	}
	return cdrPath, err
}

func InstallVim(installDir string, nvim bool, containerArch string) (string, error) {
	var vimPath string
	var err error
	if !nvim {
		vimPath, err = VIM.Install(installDir, containerArch, false)
	} else {
		if runtime.GOOS == "darwin" && containerArch == "amd64" {
			// M1 Mac で amd64 のコンテナを動かすと、なぜか AppImage が動かないので vim にフォールバック
			vimPath, err = VIM.Install(installDir, containerArch, false)
		} else {
			vimPath, err = NVIM.Install(installDir, containerArch, false)
		}
	}
	return vimPath, err
}

// start サブコマンド用のツールインストール
// 戻り値は、 devcontainerPath, cdrPath, error
func InstallStartTools(installDir string) (string, string, error) {
	var err error
	devcontainerPath, err := DEVCONTAINER.Install(installDir, "", false)
	if err != nil {
		return devcontainerPath, "", err
	}
	cdrPath, err := CDR.Install(installDir, "", false)
	if err != nil {
		return devcontainerPath, cdrPath, err
	}
	return devcontainerPath, cdrPath, nil
}

// devcontainer サブコマンド用のツールインストール
func InstallDevcontainerTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, "", false)
	return devcontainerPath, err
}

// Templates サブコマンド用のツールインストール
func InstallTemplatesTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, "", false)
	return devcontainerPath, err
}

// Stop サブコマンド用のツールインストール
func InstallStopTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, "", false)
	return devcontainerPath, err
}

// Down サブコマンド用のツールインストール
func InstallDownTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER.Install(installDir, "", false)
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
		downloadURL = fmt.Sprintf("https://github.com/mikoto2000/devcontainer.vim/releases/download/%s/devcontainer.vim-darwin-arm64", latestTagName)
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
