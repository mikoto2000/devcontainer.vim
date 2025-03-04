package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const CdrFileName = "clipboard-data-receiver"
const CdrFileNameForMac = "clipboard-data-receiver"
const cdrFileNameForWindows = "clipboard-data-receiver.exe"

// clipboard-data-receiver のダウンロード URL
const downloadURLCdrPattern = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.linux-amd64"
const downloadURLCdrPatternForMac = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.darwin-amd64"
const downloadURLCdrPatternForWindows = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.windows-amd64.exe"

const vimScriptTemplateSendToCdr = `if !has("nvim")
  function! SendToCdr(register) abort
    let text = getreg(a:register)
    let l:channelToCdr = ch_open("host.docker.internal:{{ .Port }}", {"mode": "raw"})
    call ch_sendraw(channelToCdr, l:text, {})
    call ch_close(l:channelToCdr)
  endfunction
endif`

const luaScriptTemplateSendToCdr = `function SendToCdr(register)
  local text = vim.fn.getreg(register)
  local uv = vim.loop
  local host = "host.docker.internal"
  local port = {{ .Port }}

  -- ホスト名を解決
  uv.getaddrinfo(host, nil, { socktype = "STREAM" }, function(err, res)
    if err then
      print("DNS resolution error: " .. err)
      return
    end

    local addr = res[1].addr -- 解決されたIPアドレス
    local client = uv.new_tcp()

    -- TCP接続
    client:connect(addr, port, function(connect_err)
      if connect_err then
        print("Connection error: " .. connect_err)
        return
      end

      -- データを送信
      client:write(text, function(write_err)
        if write_err then
          print("Write error: " .. write_err)
        end

        -- 接続を閉じる
        client:shutdown(function(shutdown_err)
          if shutdown_err then
            print("Shutdown error: " .. shutdown_err)
          end
          client:close()
        end)
      end)
    end)
  end)
end
`

// clipboard-data-receiver のツール情報
var CDR = func(services InstallerUseServices) Tool {

	// WSL 上で実行されているかを判定し、
	// WSL 上で実行されているなら `.exe` をダウンロード
	var cdrFileName string
	var tmpl *template.Template
	var err error
	if util.IsWsl() {
		cdrFileName = cdrFileNameForWindows
		tmpl, err = template.New("ducp").Parse(downloadURLCdrPatternForWindows)
	} else if runtime.GOOS == "darwin" {
		cdrFileName = CdrFileNameForMac
		tmpl, err = template.New("ducp").Parse(downloadURLCdrPatternForMac)
	} else {
		cdrFileName = CdrFileName
		tmpl, err = template.New("ducp").Parse(downloadURLCdrPattern)
	}
	if err != nil {
		panic(err)
	}

	// 実際に使用する cdr の構造体を返却
	return Tool{
		FileName: cdrFileName,
		CalculateDownloadURL: func(_ string) (string, error) {
			latestTagName, err := services.GetLatestReleaseFromGitHub("mikoto2000", "clipboard-data-receiver")
			if err != nil {
				return "", err
			}

			tmplParams := map[string]string{"TagName": latestTagName}
			var downloadURL strings.Builder
			err = tmpl.Execute(&downloadURL, tmplParams)
			if err != nil {
				return "", err
			}
			return downloadURL.String(), nil
		},
		installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
			return simpleInstall(downloadFunc, downloadURL, filePath)
		},
		DownloadFunc: download,
	}
}

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
	commandString := fmt.Sprintf("%s --random-port --pid-file $(wslpath -w %s) --port-file $(wslpath -w %s)", cdrPath, pidFile, portFile)
	fmt.Println(commandString)
	cdrRunCommand := exec.Command("sh", "-c", commandString)
	var stdout strings.Builder
	cdrRunCommand.Stdout = &stdout
	err := cdrRunCommand.Start()
	if err != nil {
		return 0, 0, err
	}

	// clipboard-data-receiver の出力を待つ

	// PID ファイル出力まで待つ
	// 現状 Windows で PID ファイルとポートファイルの後始末ができないので、
	// とりあえず 1 秒待てば生成されるでしょうという感じで待っている。
	var pid int
	for i := 0; i < 10; i++ {
		pidFileContentBytes, err := os.ReadFile(pidFile)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		pid, err = strconv.Atoi(string(pidFileContentBytes))
		if err != nil {
			return 0, 0, err
		}

		break
	}

	// ポートファイル出力まで待つ
	var port int
	for i := 0; i < 10; i++ {
		portFileContentBytes, err := os.ReadFile(portFile)
		if err != nil {
			// ポートファイル出力まで待つ
			time.Sleep(1 * time.Second)
			continue
		}

		port, err = strconv.Atoi(string(portFileContentBytes))
		if err != nil {
			return 0, 0, err
		}

		break
	}

	return pid, port, nil
}

func KillCdr(pid int) error {
	if util.IsWsl() {
		commandString := fmt.Sprintf("Stop-Process -Id %d -Force", pid)
		fmt.Printf("Stop clipboard-data-receiver: %s\n", commandString)
		cdrRunCommand := exec.Command("powershell.exe", "-Command", commandString)
		err := cdrRunCommand.Start()
		if err != nil {
			return err
		}
	} else {
		process, err := os.FindProcess(pid)
		if err != nil {
			return err
		}
		err = process.Kill()
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateSendToTCP(configDir string, port int, nvim bool) (string, error) {
	// SendToTcp.vim の文字列を組み立て
	var tmpl *template.Template
	var err error
	if nvim {
		tmpl, err = template.New("SendToTcp").Parse(luaScriptTemplateSendToCdr)
		if err != nil {
			return "", err
		}
	} else {
		tmpl, err = template.New("SendToTcp").Parse(vimScriptTemplateSendToCdr)
		if err != nil {
			return "", err
		}
	}

	tmplParams := map[string]int{"Port": port}
	var sendToTCPString strings.Builder
	err = tmpl.Execute(&sendToTCPString, tmplParams)
	if err != nil {
		return "", err
	}

	// ファイルに出力
	var sendToTCP string
	if nvim {
		sendToTCP = filepath.Join(configDir, "SendToTcp.lua")
	} else {
		sendToTCP = filepath.Join(configDir, "SendToTcp.vim")
	}
	err = os.WriteFile(sendToTCP, []byte(sendToTCPString.String()), 0666)
	if err != nil {
		return "", err
	}

	// 作成したファイルを返却
	return sendToTCP, nil
}
