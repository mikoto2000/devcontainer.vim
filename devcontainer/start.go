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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

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
	const listMarkers = `dir="` + portForwarderMarkerDir + `"
[ -e "$dir" ] || exit 0
if ! [ -d "$dir" ] || ! [ -r "$dir" ] || ! [ -x "$dir" ]; then
	printf 'Cannot read port-forwarder marker directory: %s\n' "$dir" >&2
	exit 1
fi
for marker in "$dir"/*; do
	[ -f "$marker" ] || continue
	printf '%s\000' "${marker##*/}" || exit
done
exit 0`
	lspfOut, err := docker.Exec(containerID, "sh", "-c", listMarkers)
	if err != nil {
		return nil, fmt.Errorf("list port-forwarder markers: %w", err)
	}

	forwardConfigs := strings.Split(lspfOut, "\x00")
	forwardConfigs = util.RemoveEmptyString(forwardConfigs)
	return forwardConfigs, nil
}

type portForwardingSession struct {
	ctx     context.Context
	workers sync.WaitGroup
}

func (s *portForwardingSession) startPortForwarders(containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath string, forwardConfigs []ForwardConfig) error {
	fmt.Println("Start port-forwarder in container.")

	for _, fc := range forwardConfigs {
		target := net.JoinHostPort(fc.Host, fc.Port)
		args := []string{"exec", "--container-id", containerID, "--workspace-folder", workspaceFolder,
			"--override-config", configFilePath, "/port-forwarder", "-l", "0.0.0.0:0", "-f", target}
		if err := s.startPortForwarder(containerID, containerIp, fc.Port, target, devcontainerPath, args); err != nil {
			return err
		}
	}
	return nil
}

func (s *portForwardingSession) startPortForwarder(containerID, containerIP, port, target, command string, args []string) error {
	listener, err := listenForPortForwarding(s.ctx, port)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(s.ctx)
	started := false
	defer func() {
		if !started {
			cancel()
			listener.Close()
		}
	}()

	fmt.Printf("Start port-forwarder for %s.\n", target)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = os.Stderr
	cmd.WaitDelay = 2 * time.Second
	if runtime.GOOS != "windows" {
		cmd.Cancel = func() error {
			return cmd.Process.Signal(os.Interrupt)
		}
	}

	// Let os/exec manage its own pipe so Wait can reap the child while we read stdout.
	portOut, output := io.Pipe()
	cmd.Stdout = output
	if err := cmd.Start(); err != nil {
		portOut.Close()
		output.Close()
		return fmt.Errorf("start port-forwarder for %s: %w", target, err)
	}
	ready := make(chan string, 1)
	s.workers.Go(func() {
		defer portOut.Close()
		scanner := bufio.NewScanner(portOut)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			listenPort, err := parseForwardPort(line)
			if err == nil && listenPort != 0 {
				ready <- strconv.FormatUint(listenPort, 10)
				break
			}
			fmt.Fprintf(os.Stderr, "Invalid port-forwarder listen port: %q\n", line)
		}
		io.Copy(io.Discard, portOut)
	})
	done := make(chan struct{})
	var exitErr error
	s.workers.Go(func() {
		exitErr = cmd.Wait()
		if exitErr == nil {
			exitErr = errors.New("process exited")
		}
		output.Close()
		close(done)
		cancel()
	})

	startup, stopStartup := context.WithTimeout(s.ctx, 30*time.Second)
	defer stopStartup()
	var destination string
	select {
	case <-startup.Done():
		return fmt.Errorf("wait for port-forwarder for %s: %w", target, startup.Err())
	case <-done:
		return fmt.Errorf("port-forwarder for %s exited during startup: %w", target, exitErr)
	case listenPort := <-ready:
		destination = net.JoinHostPort(containerIP, listenPort)
	}
	// Expand HOME in a fixed script, with the marker name passed as data.
	const writeMarker = `dir="` + portForwarderMarkerDir + `"; mkdir -p "$dir" && touch "$dir/$1"`
	if _, err := docker.Exec(containerID, "sh", "-c", writeMarker, "port-forwarder-marker", target+"_"+destination); err != nil {
		return fmt.Errorf("create port-forwarder marker for %s: %w", target, err)
	}
	select {
	case <-done:
		return fmt.Errorf("port-forwarder for %s exited during startup: %w", target, exitErr)
	default:
	}

	fmt.Printf("port-forwarder started: %s %s\n", destination, target)
	started = true
	s.workers.Go(func() {
		defer cancel()
		servePortForwarding(ctx, listener, destination)
		select {
		case <-done:
			if s.ctx.Err() == nil {
				fmt.Fprintf(os.Stderr, "port-forwarder for %s exited: %v\n", target, exitErr)
			}
		default:
		}
	})
	return nil
}

func listenForPortForwarding(ctx context.Context, port string) (net.Listener, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	addr := net.JoinHostPort("0.0.0.0", port)
	config := portForwardListenConfig()
	listener, err := config.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for port forwarding on %s: %w", addr, err)
	}
	return listener, nil
}

func servePortForwarding(ctx context.Context, listener net.Listener, destination string) {
	addr := listener.Addr()
	if err := util.ServeForwarding(ctx, listener, destination); err != nil {
		fmt.Fprintf(os.Stderr, "port forwarding on %s failed: %v\n", addr, err)
	}
}

func (s *portForwardingSession) restorePortForwarders(containerIp string, forwardConfigs []string) error {
	for _, forwardConfig := range forwardConfigs {
		containerSrcPort, containerDestPort, err := parsePortForwarderMarker(forwardConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skip invalid port-forwarder marker: %s: %v\n", forwardConfig, err)
			continue
		}

		listener, err := listenForPortForwarding(s.ctx, containerSrcPort)
		if err != nil {
			return err
		}
		destination := net.JoinHostPort(containerIp, containerDestPort)
		fmt.Printf("listen: %s, forward: %s.\n", listener.Addr(), destination)
		s.workers.Go(func() { servePortForwarding(s.ctx, listener, destination) })
	}
	return nil
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
func setupPortForwarding(ctx context.Context, containerID, devcontainerPath, workspaceFolder, configFilePath string) (func(), error) {
	configuration, err := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder, "--override-config", configFilePath)
	if err != nil {
		return nil, err
	}
	forwardConfigs, err := GetForwardPorts(configuration)
	if err != nil {
		return nil, err
	}
	if len(forwardConfigs) == 0 {
		return func() {}, nil
	}

	// コンテナの IP アドレスを取得
	containerIp, err := docker.Exec(containerID, "hostname", "-i")
	if err != nil {
		return nil, errors.New("コンテナ上での hostname 実行に失敗しました。コンテナに hostname コマンドがインストールされている必要があります")
	}
	containerIp = strings.TrimSpace(containerIp)
	if net.ParseIP(containerIp) == nil {
		return nil, fmt.Errorf("invalid container IP address: %q", containerIp)
	}

	portForwarders, err := listRunningPortForwarders(containerID)
	if err != nil {
		return nil, err
	}
	markers, err := listPortForwarderMarkers(containerID)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	session := &portForwardingSession{ctx: ctx}
	stop := func() {
		cancel()
		session.workers.Wait()
	}
	if len(portForwarders) > 0 && len(markers) > 0 {
		fmt.Println("port-forwarder already running.")
		err = session.restorePortForwarders(containerIp, markers)
	} else {
		if len(portForwarders) > 0 {
			fmt.Fprintf(os.Stderr, "port-forwarder process exists but marker files are missing. Restart port-forwarder setup.\n")
		}
		err = session.startPortForwarders(containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath, forwardConfigs)
	}
	if err != nil {
		stop()
		return nil, err
	}
	return stop, nil
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

	if len(args) == 0 {
		return errors.New("missing workspace folder")
	}

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
		stopForwarding, err := setupPortForwarding(context.Background(), containerID, devcontainerPath, workspaceFolder, configFilePath)
		if err != nil {
			return err
		}
		defer stopForwarding()
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
