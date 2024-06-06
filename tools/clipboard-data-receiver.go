package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
	"time"

	"github.com/mikoto2000/devcontainer.vim/util"
)

const CDR_FILE_NAME = "clipboard-data-receiver"
const CDR_FILE_NAME_FOR_WINDOWS = "clipboard-data-receiver.exe"

// clipboard-data-receiver のダウンロード URL
const DOWNLOAD_URL_CDR_PATTERN = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.linux-amd64"
const DOWNLOAD_URL_CDR_PATTERN_FOR_WINDOWS = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.linux-amd64.exe"

// clipboard-data-receiver のツール情報
var CDR Tool = func() Tool {

	// WSL 上で実行されているかを判定し、
	// WSL 上で実行されているなら `.exe` をダウンロード
	var cdrFileName string
	var tmpl *template.Template
	var err error
	if util.IsWsl() {
		cdrFileName = CDR_FILE_NAME_FOR_WINDOWS
		tmpl, err = template.New("ducp").Parse(DOWNLOAD_URL_CDR_PATTERN)
	} else {
		cdrFileName = CDR_FILE_NAME
		tmpl, err = template.New("ducp").Parse(DOWNLOAD_URL_CDR_PATTERN_FOR_WINDOWS)
	}
	if err != nil {
		panic(err)
	}

	// 実際に使用する cdr の構造体を返却
	return Tool{
		FileName: cdrFileName,
		CalculateDownloadUrl: func() string {
			latestTagName, err := util.GetLatestReleaseFromGitHub("mikoto2000", "clipboard-data-receiver")
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
}()

// clipboard-data-receiver を起動
// pid ファイル、 port ファイルを configFileDir へ保存する。
func RunCdr(cdrPath string, configFileDir string) (int, int, error) {
	// configFileDir から pid ファイルと port ファイルのパスを組み立てる
	pidFile := filepath.Join(configFileDir, "pid")
	portFile := filepath.Join(configFileDir, "port")

	// Windows 判定
	if runtime.GOOS == "windows" {
		return runCdrForNative(cdrPath, pidFile, portFile)
	} else {
		if util.IsWsl() {
			return runCdrForWsl(cdrPath, pidFile, portFile)
		} else {
			return runCdrForNative(cdrPath, pidFile, portFile)
		}
	}
}

// clipboard-data-receiver を、 WSL でない環境で実行する場合の処理
func runCdrForNative(cdrPath string, pidFile string, portFile string) (int, int, error) {
	fmt.Println("\""+cdrPath+"\"", "--pid-file", pidFile, "--port-file", portFile, "--random-port")
	cdrRunCommand := exec.Command(cdrPath, "--pid-file", pidFile, "--port-file", portFile, "--random-port")
	var stdout strings.Builder
	cdrRunCommand.Stdout = &stdout
	err := cdrRunCommand.Start()
	if err != nil {
		return 0, 0, err
	}

	// clipboard-data-receiver の出力を待つ
	// タイムアウト 10 秒
	var pid, port int
	for i := 0; i < 10; i++ {
		pid, _, port, err = GetProcessInfo(stdout.String())
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		} else {
			break
		}
	}

	return pid, port, nil
}

func runCdrForWsl(cdrPath string, pidFile string, portFile string) (int, int, error) {
	// clipboard-data-receiver.exe を実行
	panic("not implement")
}
