package devcontainer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const portForwarderMarkerDir = "$HOME/.config/devcontainer.vim/pf"

type DevcontainerStartUseService interface {
	StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePathForDevcontainer string) error
}

type DefaultDevcontainerStartUseService struct{}

func (s DefaultDevcontainerStartUseService) StartVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configDirForDevcontainer string) error {
	return startVim(containerID, devcontainerPath, workspaceFolder, vimFileName, tmuxFileName, sendToTCP, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configDirForDevcontainer)
}

var devcontainreArgsPrefix = []string{"up"}

// devcontainer up でコンテナを起動し、コンテナIDを返す
func startDevcontainer(devcontainerPath string, args []string, configFilePath string, workspaceFolder string) (string, error) {
	// 末尾以外のものはそのまま `devcontainer up` への引数として渡す
	userArgs := args[0 : len(args)-1]
	userArgs = append(userArgs, "--override-config", configFilePath, "--workspace-folder", workspaceFolder)
	devcontainerArgs := append(devcontainreArgsPrefix, userArgs...)
	fmt.Printf("run container: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerArgs, "\" \""))

	dockerRunCommand := exec.Command(devcontainerPath, devcontainerArgs...)
	dockerRunCommand.Stderr = os.Stderr

	stdout, err := dockerRunCommand.Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		return "", err
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		return "", err
	}

	containerID := upCommandResult.ContainerID
	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	return containerID, nil
}

// devcontainer用のclipboard-data-receiverを起動する
func startClipboardReceiverForDevcontainer(cdrPath, configDirForDevcontainer string) (int, int, error) {
	pid, port, err := tools.RunCdr(cdrPath, configDirForDevcontainer)
	if err != nil {
		return 0, 0, err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)
	return pid, port, nil
}

func listRunningPortForwarders(containerID string) ([]string, error) {
	psOut, err := docker.Exec(containerID, "sh", "-c", "grep --files-with-matches port-forwarder /proc/*/comm || true")
	if err != nil {
		return nil, err
	}

	portForwarders := strings.Split(strings.TrimSpace(psOut), "\n")
	portForwarders = util.RemoveEmptyString(portForwarders)

	fmt.Printf("Running port-forwarders: %s\n", portForwarders)
	return portForwarders, nil
}

func listPortForwarderMarkers(containerID string) ([]string, error) {
	lspfOut, err := docker.Exec(containerID, "sh", "-c", `ls --zero "`+portForwarderMarkerDir+`" 2>/dev/null || true`)
	if err != nil {
		return nil, err
	}

	forwardConfigs := strings.Split(lspfOut, "\x00")
	forwardConfigs = util.RemoveEmptyString(forwardConfigs)
	return forwardConfigs, nil
}

func startPortForwarders(ctx context.Context, containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath string, forwardConfigs []ForwardConfig) error {
	fmt.Println("Start port-forwarder in container.")

	// 解釈した forwardPort ごとに port-forwarder を起動する
	for _, fc := range forwardConfigs {
		target := net.JoinHostPort(fc.Host, fc.Port)
		args := []string{"exec", "--container-id", containerID, "--workspace-folder", workspaceFolder,
			"--override-config", configFilePath, "/port-forwarder", "-l", "0.0.0.0:0", "-f", target}
		fmt.Printf("Start port-forwarder for %s.\n", target)
		dockerExecPortForwarder := exec.CommandContext(ctx, devcontainerPath, args...)
		portOut, err := dockerExecPortForwarder.StdoutPipe()
		if err != nil {
			return err
		}

		dockerExecPortForwarder.Cancel = func() error {
			fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
			return dockerExecPortForwarder.Process.Signal(os.Interrupt)
		}

		err = dockerExecPortForwarder.Start()
		if err != nil {
			return err
		}

		go func() {
			defer dockerExecPortForwarder.Wait()
			reader := bufio.NewReader(portOut)
			for {
				port, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						fmt.Println("Error reading from stdout:", err)
					}
					break
				}
				listenPort, err := parseForwardPort(strings.TrimSpace(port))
				if err != nil || listenPort == 0 {
					fmt.Fprintf(os.Stderr, "Invalid port-forwarder listen port: %q\n", strings.TrimSpace(port))
					continue
				}
				destination := net.JoinHostPort(containerIp, strconv.FormatUint(listenPort, 10))
				fmt.Printf("port-forwarder started: %s %s\n", destination, target)

				// Expand HOME in a fixed script, with the marker name passed as data.
				const writeMarker = `dir="` + portForwarderMarkerDir + `"; mkdir -p "$dir" && touch "$dir/$1"`
				_, err = docker.Exec(containerID, "sh", "-c", writeMarker, "port-forwarder-marker", target+"_"+destination)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error creating port-forwarder marker file: %v\n", err)
					continue
				}

				util.StartForwarding("0.0.0.0:"+fc.Port, destination)
			}
		}()
	}

	return nil
}

func restorePortForwarders(containerIp string, forwardConfigs []string) {
	for _, forwardConfig := range forwardConfigs {
		containerSrcPort, containerDestPort, err := parsePortForwarderMarker(forwardConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skip invalid port-forwarder marker: %s: %v\n", forwardConfig, err)
			continue
		}

		go func() {
			destination := net.JoinHostPort(containerIp, containerDestPort)
			fmt.Printf("listen: %s, forward: %s.\n", "0.0.0.0:"+containerSrcPort, destination)
			util.StartForwarding("0.0.0.0:"+containerSrcPort, destination)
		}()
	}
}

func parsePortForwarderMarker(forwardConfig string) (string, string, error) {
	splitedForwardConfig := strings.Split(forwardConfig, "_")
	if len(splitedForwardConfig) != 2 {
		return "", "", errors.New("marker must contain source and destination")
	}

	_, srcPort, err := net.SplitHostPort(splitedForwardConfig[0])
	if err != nil {
		return "", "", errors.New("source marker must contain host and port")
	}

	_, destPort, err := net.SplitHostPort(splitedForwardConfig[1])
	if err != nil {
		return "", "", errors.New("destination marker must contain host and port")
	}

	return srcPort, destPort, nil
}

// port-forwardingの設定を行う
func setupPortForwarding(ctx context.Context, containerID, devcontainerPath, workspaceFolder, configFilePath string) error {
	configuration, err := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder, "--override-config", configFilePath)
	if err != nil {
		return err
	}
	forwardConfigs, err := GetForwardPorts(configuration)
	if err != nil || len(forwardConfigs) == 0 {
		return err
	}

	// コンテナの IP アドレスを取得
	containerIp, err := docker.Exec(containerID, "hostname", "-i")
	if err != nil {
		return errors.New("コンテナ上での hostname 実行に失敗しました。コンテナに hostname コマンドがインストールされている必要があります")
	}
	containerIp = strings.TrimSpace(containerIp)
	if net.ParseIP(containerIp) == nil {
		return fmt.Errorf("invalid container IP address: %q", containerIp)
	}

	portForwarders, err := listRunningPortForwarders(containerID)
	if err != nil {
		return err
	}
	markers, err := listPortForwarderMarkers(containerID)
	if err != nil {
		return err
	}

	if len(portForwarders) == 0 {
		return startPortForwarders(ctx, containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath, forwardConfigs)
	}

	if len(markers) == 0 {
		fmt.Fprintf(os.Stderr, "port-forwarder process exists but marker files are missing. Restart port-forwarder setup.\n")
		return startPortForwarders(ctx, containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath, forwardConfigs)
	}

	fmt.Println("port-forwarder already running.")
	restorePortForwarders(containerIp, markers)

	return nil
}

// devcontainer でコンテナを立ち上げ、 Vim を転送し、実行する。
// 既存実装の都合上、configFilePath から configDirForDevcontainer を抽出している
func Start(
	services DevcontainerStartUseService,
	args []string,
	devcontainerPath string,
	noCdr bool,
	noPf bool,
	noTmux bool,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	shell string,
	configFilePath string,
	vimrc string) error {

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// 1. devcontainer up でコンテナを起動
	containerID, err := startDevcontainer(devcontainerPath, args, configFilePath, workspaceFolder)
	if err != nil {
		return err
	}

	// 2. コンテナアーキテクチャを取得
	containerArch, err := getContainerArch(containerID)
	if err != nil {
		return err
	}

	// 3. port-forwarderをインストール
	if !noPf {
		if err := installPortForwarder(containerID, vimInstallDir, containerArch); err != nil {
			return err
		}
	}

	// 4. clipboard-data-receiverを起動
	port := 0
	configDirForDevcontainer := filepath.Dir(configFilePath)
	if !noCdr {
		_, port, err = startClipboardReceiverForDevcontainer(cdrPath, configDirForDevcontainer)
		if err != nil {
			return err
		}
	}

	// 5. port-forwardingの設定
	if !noPf {
		pfCtx, pfCancel := context.WithCancel(context.Background())
		defer pfCancel()
		err = setupPortForwarding(pfCtx, containerID, devcontainerPath, workspaceFolder, configFilePath)
		if err != nil {
			return err
		}
	}

	// 6. Vimの検出とインストール
	vimFileName, useSystemVim, err := setupVim(containerID, vimInstallDir, nvim, containerArch)
	if err != nil {
		return err
	}

	tmuxFileName := ""
	useSystemTmux := false
	if !noTmux {
		tmuxFileName, useSystemTmux, err = setupTmux(containerID, vimInstallDir, containerArch)
		if err != nil {
			return err
		}
	}

	// 7. Vimファイルの転送
	sendToTCP, err := transferVimFiles(containerID, configDirForDevcontainer, vimrc, noCdr, port, vimFileName == "nvim")
	if err != nil {
		return err
	}

	// 8. コンテナへ接続
	err = services.StartVim(containerID, devcontainerPath, workspaceFolder, vimFileName, tmuxFileName, sendToTCP, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configDirForDevcontainer)
	if err != nil {
		return err
	}

	// コンテナ停止は別途 down コマンドで行う
	return nil
}

// コンテナへ接続
// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`
func startVim(containerID string, devcontainerPath string, workspaceFolder string, vimFileName string, tmuxFileName string, sendToTCP string, containerArch string, useSystemVim bool, useSystemTmux bool, noTmux bool, shell string, configFilePathForDevcontainer string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sendToTCPName := filepath.Base(sendToTCP)
	devcontainerStartVimArgs, err := devcontainerStartVimArgs(containerID, workspaceFolder, vimFileName, tmuxFileName, sendToTCPName, containerArch, useSystemVim, useSystemTmux, noTmux, shell, configFilePathForDevcontainer)
	if err != nil {
		return err
	}
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerStartVimArgs, "\" \""))
	dockerExec := createStartVimCommand(ctx, devcontainerPath, devcontainerStartVimArgs)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	err = dockerExec.Run()
	if err != nil {
		return err
	} else {
		return nil
	}
}

func createStartVimCommand(ctx context.Context, devcontainerPath string, devcontainerStartVimArgs []string) *exec.Cmd {
	if util.IsWsl() && util.IsExistsCommand("script") {
		scriptArgs := []string{"-qefc", shellQuote(devcontainerPath)}
		for _, arg := range devcontainerStartVimArgs {
			scriptArgs[1] += " " + shellQuote(arg)
		}
		scriptArgs = append(scriptArgs, "/dev/null")
		return exec.CommandContext(ctx, "script", scriptArgs...)
	}

	return exec.CommandContext(ctx, devcontainerPath, devcontainerStartVimArgs...)
}

func shellQuote(arg string) string {
	return "'" + strings.ReplaceAll(arg, "'", `'\''`) + "'"
}
