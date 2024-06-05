//go:build windows

package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const CDR_FILE_NAME = "clipboard-data-receiver.exe"

// clipboard-data-receiver のダウンロード URL
const DOWNLOAD_URL_CDR_PATTERN = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.windows-amd64.exe"

func RunCdrForDocker(cdrPath string, configFileDir string) (int, int, error) {
	// configFileDir から pid ファイルと port ファイルのパスを組み立てる
	pidFile := filepath.Join(configFileDir, "pid")
	portFile := filepath.Join(configFileDir, "port")
	return runCdr(cdrPath, pidFile, portFile)
}

func RunCdrForDevcontainer(cdrPath string, configFileDir string) (int, int, error) {
	// configFileDir から pid ファイルと port ファイルのパスを組み立てる
	pidFile := filepath.Join(configFileDir, "pid")
	portFile := filepath.Join(configFileDir, "port")
	return runCdr(cdrPath, pidFile, portFile)
}

func runCdr(cdrPath string, pidFile string, portFile string) (int, int, error) {
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
