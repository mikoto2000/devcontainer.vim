package tools

import (
	"debug/buildinfo"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"slices"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

type InstallerUseServices interface {
	GetLatestReleaseFromGitHub(owner string, repository string) (string, error)
	Download(downloadURL string, destPath string) error
}

type DefaultInstallerUseServices struct{}

func (s DefaultInstallerUseServices) GetLatestReleaseFromGitHub(owner string, repository string) (string, error) {
	return util.GetLatestReleaseFromGitHub(owner, repository)
}

func (s DefaultInstallerUseServices) Download(downloadURL string, destPath string) error {
	return download(downloadURL, destPath)
}

// ツール情報
type Tool struct {
	FileName             string
	CalculateDownloadURL func(containerArch string) (string, error)
	installFunc          func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error)
	DownloadFunc         func(downloadURL string, destPath string) error
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
		err = stageFile(filePath, func(stagedPath string) error {
			_, err := t.installFunc(t.DownloadFunc, downloadURL, stagedPath, containerArch)
			return err
		}, os.Rename)
		return filePath, err
	}
}

// Keep the destination in place until the new file is ready to publish.
func stageFile(destPath string, prepare func(string) error, publish func(string, string) error) error {
	dir, err := os.MkdirTemp(filepath.Dir(destPath), ".devcontainer.vim-stage-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	stagedPath := filepath.Join(dir, filepath.Base(destPath))
	if err := prepare(stagedPath); err != nil {
		return err
	}
	return publish(stagedPath, destPath)
}

// 単純なファイル配置でインストールが完了するもののインストール処理。
//
// downloadURL からファイルをダウンロードし、 installDir に fileName とう名前で配置する。
func simpleInstall(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string) (string, error) {

	// ツールのダウンロード
	err := downloadFunc(downloadURL, filePath)
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected HTTP status %s", filepath.Base(destPath), resp.Status)
	}
	err = stageFile(destPath, func(stagedPath string) error {
		out, err := os.Create(stagedPath)
		if err != nil {
			return err
		}
		var body io.Reader = resp.Body
		if resp.ContentLength > 0 {
			body = io.TeeReader(body, &ProgressWriter{Total: resp.ContentLength})
		}
		size, copyErr := io.Copy(out, body)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if size == 0 {
			return fmt.Errorf("download %s: empty response body", filepath.Base(destPath))
		}
		return nil
	}, os.Rename)
	if err != nil {
		return err
	}

	fmt.Printf(" done. \n")

	return nil
}

// run サブコマンド用のツールインストール
func InstallRunTools(installDir string, nvim bool) (string, error) {
	var err error
	cdrPath, err := CDR(DefaultInstallerUseServices{}).Install(installDir, "", false)
	if err != nil {
		return cdrPath, err
	}
	return cdrPath, err
}

func InstallVim(installDir string, nvim bool, containerArch string) (string, error) {
	// Use static Vim for ARM containers and for amd64 containers on macOS,
	// where this setup does not use the Neovim AppImage.
	armContainer := containerArch == "aarch64" || containerArch == "arm64"
	macStaticVim := runtime.GOOS == "darwin" && (containerArch == "amd64" || containerArch == "x86_64")
	if !nvim || armContainer || macStaticVim {
		return VIM(DefaultInstallerUseServices{}).Install(installDir, containerArch, false)
	}
	return NVIM(DefaultInstallerUseServices{}).Install(installDir, containerArch, false)
}

func InstallTmux(installDir string, containerArch string) (string, error) {
	return Tmux(DefaultInstallerUseServices{}).Install(installDir, containerArch, false)
}

// start サブコマンド用のツールインストール
// 戻り値は、 devcontainerPath, cdrPath, error
func InstallStartTools(services InstallerUseServices, installDir string) (string, string, error) {
	var err error
	devcontainerPath, err := DEVCONTAINER(services).Install(installDir, "", false)
	if err != nil {
		return devcontainerPath, "", err
	}
	cdrPath, err := CDR(services).Install(installDir, "", false)
	if err != nil {
		return devcontainerPath, cdrPath, err
	}
	return devcontainerPath, cdrPath, nil
}

// devcontainer サブコマンド用のツールインストール
func InstallDevcontainerTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER(DefaultInstallerUseServices{}).Install(installDir, "", false)
	return devcontainerPath, err
}

// Templates サブコマンド用のツールインストール
func InstallTemplatesTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER(DefaultInstallerUseServices{}).Install(installDir, "", false)
	return devcontainerPath, err
}

// Stop サブコマンド用のツールインストール
func InstallStopTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER(DefaultInstallerUseServices{}).Install(installDir, "", false)
	return devcontainerPath, err
}

// Down サブコマンド用のツールインストール
func InstallDownTools(installDir string) (string, error) {
	devcontainerPath, err := DEVCONTAINER(DefaultInstallerUseServices{}).Install(installDir, "", false)
	return devcontainerPath, err
}

// SelfUpdate downloads the latest release of devcontainer.vim from GitHub and replaces the current binary
func SelfUpdate(services InstallerUseServices) error {
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}
	return selfUpdate(services, executablePath)
}

func selfUpdate(services InstallerUseServices, executablePath string) error {
	// Get the latest release tag name from GitHub
	latestTagName, err := services.GetLatestReleaseFromGitHub("mikoto2000", "devcontainer.vim")
	if err != nil {
		return err
	}

	// Construct the download URL for the latest release
	downloadURL := fmt.Sprintf("https://github.com/mikoto2000/devcontainer.vim/releases/download/%s/devcontainer.vim-%s-%s", latestTagName, runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		downloadURL = downloadURL + ".exe"
	}

	if err := updateExecutable(services.Download, downloadURL, executablePath); err != nil {
		return err
	}
	fmt.Println("devcontainer.vim has been updated to the latest version.")
	return nil
}

func updateExecutable(downloadFunc func(string, string) error, downloadURL, executablePath string) error {
	return stageFile(executablePath, func(stagedPath string) error {
		if _, err := simpleInstall(downloadFunc, downloadURL, stagedPath); err != nil {
			return err
		}
		info, err := buildinfo.ReadFile(stagedPath)
		if err != nil {
			return fmt.Errorf("read update executable build info: %w", err)
		}
		if info.Path != "github.com/mikoto2000/devcontainer.vim/v3" ||
			!slices.Contains(info.Settings, debug.BuildSetting{Key: "GOOS", Value: runtime.GOOS}) ||
			!slices.Contains(info.Settings, debug.BuildSetting{Key: "GOARCH", Value: runtime.GOARCH}) {
			return fmt.Errorf("update is not a devcontainer.vim executable for %s/%s", runtime.GOOS, runtime.GOARCH)
		}
		return nil
	}, replaceExecutable)
}

func replaceExecutable(stagedPath, executablePath string) error {
	if runtime.GOOS == "windows" {
		// Windows cannot replace an executable while it is running.
		return replaceExecutableWithBackup(stagedPath, executablePath, os.Rename)
	}
	return os.Rename(stagedPath, executablePath)
}

func replaceExecutableWithBackup(stagedPath, executablePath string, rename func(string, string) error) error {
	backup, err := os.CreateTemp(filepath.Dir(executablePath), ".devcontainer.vim-old-*.exe")
	if err != nil {
		return err
	}
	backupPath := backup.Name()
	if err := backup.Close(); err != nil {
		os.Remove(backupPath)
		return err
	}
	if err := rename(executablePath, backupPath); err != nil {
		os.Remove(backupPath)
		return fmt.Errorf("back up executable: %w", err)
	}

	if err := rename(stagedPath, executablePath); err != nil {
		if restoreErr := rename(backupPath, executablePath); restoreErr != nil {
			return fmt.Errorf("replace executable: %w; could not restore from %q: %w", err, backupPath, restoreErr)
		}
		return fmt.Errorf("replace executable: %w", err)
	}

	if err := os.Remove(backupPath); err != nil {
		fmt.Fprintf(os.Stderr, "Updated executable; previous version remains at %q: %v\n", backupPath, err)
	}
	return nil
}
