package tools

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const CdrFileName = "clipboard-data-receiver"
const CdrFileNameForMac = "clipboard-data-receiver"
const cdrFileNameForWindows = "clipboard-data-receiver.exe"

var errCdrStartupTimeout = errors.New("clipboard-data-receiver startup timed out")

const cdrStartupTimeout = 10 * time.Second
const cdrPollInterval = time.Second

func IsCdrStartupTimeout(err error) bool {
	return errors.Is(err, errCdrStartupTimeout)
}

// clipboard-data-receiver のダウンロード URL
const downloadURLCdrPattern = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.linux-amd64"
const downloadURLCdrPatternForMac = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.darwin-amd64"
const downloadURLCdrPatternForWindows = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.windows-amd64.exe"

const vimScriptTemplateSendToCdr = `if !has("nvim")
  function! SendToCdr(register) abort
    let text = getreg(a:register)
    let l:channelToCdr = ch_open("host.docker.internal:{{ .Port }}", {"mode": "raw"})
    if ch_status(l:channelToCdr) !=# "open"
      echohl WarningMsg
      echomsg "Could not connect to clipboard-data-receiver."
      echohl None
      return
    endif
    call ch_sendraw(l:channelToCdr, l:text, {})
    call ch_close(l:channelToCdr)
  endfunction
endif`

const vimNoCdrScriptTemplateSendToCdr = `if !has("nvim")
  function! SendToCdr(register) abort
    " nop
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
const luaNoCdrScriptTemplateSendToCdr = `function SendToCdr(register)
  -- nop
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
		return Tool{
			FileName: cdrFileName,
			CalculateDownloadURL: func(_ string) (string, error) {
				return "", err
			},
			installFunc: func(downloadFunc func(downloadURL string, destPath string) error, downloadURL string, filePath string, containerArch string) (string, error) {
				return "", err
			},
			DownloadFunc: services.Download,
		}
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
		DownloadFunc: services.Download,
	}
}

// clipboard-data-receiver を起動
// pid ファイル、 port ファイルを configFileDir へ保存する。
// エラー時も PID が判明していれば、呼び出し元で停止できるよう返す。
func RunCdr(cdrPath string, configFileDir string) (int, int, error) {
	// configFileDir から pid ファイルと port ファイルのパスを組み立てる
	pidFile := filepath.Join(configFileDir, "pid")
	portFile := filepath.Join(configFileDir, "port")

	if runtime.GOOS != "windows" && util.IsWsl() {
		return runCdrForWsl(cdrPath, pidFile, portFile, cdrStartupTimeout, cdrPollInterval)
	}
	return runCdrForNative(cdrPath, pidFile, portFile, cdrStartupTimeout, cdrPollInterval)
}

// cdrOutputBuffer synchronizes startup polling with the os/exec output writer.
type cdrOutputBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (o *cdrOutputBuffer) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.Write(p)
}

func (o *cdrOutputBuffer) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.buf.String()
}

// clipboard-data-receiver を、 WSL でない環境で実行する場合の処理
func runCdrForNative(cdrPath, pidFile, portFile string, startupTimeout, pollInterval time.Duration) (int, int, error) {
	cdrRunCommand := exec.Command(cdrPath, "--pid-file", pidFile, "--port-file", portFile, "--random-port")
	var output cdrOutputBuffer
	cdrRunCommand.Stdout = &output
	cdrRunCommand.Stderr = &output
	err := cdrRunCommand.Start()
	if err != nil {
		return 0, 0, err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cdrRunCommand.Wait()
	}()

	deadline := time.Now().Add(startupTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pid, _, port, parseErr := GetProcessInfo(output.String())
		if parseErr == nil && pid > 0 && port > 0 {
			return pid, port, nil
		}
		if parseErr == nil {
			lastErr = fmt.Errorf("invalid clipboard-data-receiver output: pid=%d port=%d", pid, port)
		} else {
			lastErr = parseErr
		}

		select {
		case waitErr := <-waitCh:
			if waitErr != nil {
				return 0, 0, fmt.Errorf("clipboard-data-receiver exited before startup completed: %w; output: %s", waitErr, strings.TrimSpace(output.String()))
			}
			return 0, 0, fmt.Errorf("clipboard-data-receiver exited without reporting pid/port; output: %s", strings.TrimSpace(output.String()))
		case <-time.After(pollInterval):
		}
	}

	_ = cdrRunCommand.Process.Kill()
	<-waitCh
	return 0, 0, fmt.Errorf("%w: %v; output: %s", errCdrStartupTimeout, lastErr, strings.TrimSpace(output.String()))
}

func runCdrForWsl(cdrPath, pidFile, portFile string, startupTimeout, pollInterval time.Duration) (int, int, error) {
	windowsPidFile, err := wslCdrPath(pidFile)
	if err != nil {
		return 0, 0, err
	}
	windowsPortFile, err := wslCdrPath(portFile)
	if err != nil {
		return 0, 0, err
	}

	cmd := exec.Command(cdrPath, "--random-port", "--pid-file", windowsPidFile, "--port-file", windowsPortFile)
	var output cdrOutputBuffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		return 0, 0, err
	}
	done := make(chan struct{})
	var waitErr error
	go func() {
		waitErr = cmd.Wait()
		close(done)
	}()

	pid, err := waitForWslCdrFile(pidFile, done, startupTimeout, pollInterval)
	if err == nil {
		var port int
		port, err = waitForWslCdrFile(portFile, done, startupTimeout, pollInterval)
		if err == nil {
			select {
			case <-done:
				if waitErr == nil {
					return pid, port, nil
				}
				err = fmt.Errorf("clipboard-data-receiver exited during startup")
			default:
				return pid, port, nil
			}
		}
	}

	select {
	case <-done:
		err = errors.Join(err, waitErr)
	default:
		if killErr := cmd.Process.Kill(); killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return pid, 0, errors.Join(err, fmt.Errorf("stop clipboard-data-receiver: %w", killErr))
		}
		<-done
	}
	return 0, 0, fmt.Errorf("clipboard-data-receiver startup: %w; output: %s", err, strings.TrimSpace(output.String()))
}

func wslCdrPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("wslpath", "-w", abs)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("convert CDR path %q: %w", path, err)
	}
	converted := strings.TrimRight(string(out), "\r\n")
	if converted == "" {
		return "", fmt.Errorf("wslpath returned no path for %q", path)
	}
	return converted, nil
}

func waitForWslCdrFile(path string, done <-chan struct{}, startupTimeout, pollInterval time.Duration) (int, error) {
	timeout := time.NewTimer(startupTimeout)
	defer timeout.Stop()
	poll := time.NewTicker(pollInterval)
	defer poll.Stop()
	read := func() (int, error) {
		data, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		if err != nil {
			return 0, err
		}
		value := strings.TrimSpace(string(data))
		if value == "" {
			return 0, nil
		}
		return strconv.Atoi(value)
	}
	for {
		value, err := read()
		if err != nil {
			return 0, fmt.Errorf("read CDR startup file %q: %w", path, err)
		}
		if value > 0 {
			return value, nil
		}
		select {
		case <-done:
			// The child may have written the file just before exiting.
			value, err := read()
			if err != nil {
				return 0, fmt.Errorf("read CDR startup file %q: %w", path, err)
			}
			if value > 0 {
				return value, nil
			}
			return 0, fmt.Errorf("clipboard-data-receiver exited before %q was populated", path)
		case <-timeout.C:
			return 0, fmt.Errorf("%w: %q was not populated", errCdrStartupTimeout, path)
		case <-poll.C:
		}
	}
}

func KillCdr(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid clipboard-data-receiver PID %d: must be positive", pid)
	}
	if util.IsWsl() {
		commandString := fmt.Sprintf("Stop-Process -Id %d -Force", pid)
		fmt.Printf("Stop clipboard-data-receiver: %s\n", commandString)
		cdrRunCommand := exec.Command("powershell.exe", "-Command", commandString)
		return cdrRunCommand.Run()
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

func CreateSendToTCP(configDir string, port int, noCdr bool, nvim bool) (string, error) {
	// SendToTcp.vim の文字列を組み立て
	var tmpl *template.Template
	var err error
	if !noCdr {
		// cdr 有効
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
	} else {
		// cdr 無効
		if nvim {
			tmpl, err = template.New("SendToTcp").Parse(luaNoCdrScriptTemplateSendToCdr)
			if err != nil {
				return "", err
			}
		} else {
			tmpl, err = template.New("SendToTcp").Parse(vimNoCdrScriptTemplateSendToCdr)
			if err != nil {
				return "", err
			}
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
	// Docker copies this script for the container editor, which may run as a non-root user.
	err = os.WriteFile(sendToTCP, []byte(sendToTCPString.String()), 0644)
	if err != nil {
		return "", err
	}

	// 作成したファイルを返却
	return sendToTCP, nil
}
