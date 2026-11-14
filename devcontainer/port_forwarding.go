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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/internal/portforward"
)

type portForwardingSession struct {
	ctx     context.Context
	workers sync.WaitGroup
}

func (s *portForwardingSession) startPortForwarders(containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath string, forwardConfigs []ForwardConfig) error {
	fmt.Println("Start port-forwarder in container.")

	for _, fc := range forwardConfigs {
		target := net.JoinHostPort(fc.Host, fc.Port)
		args := []string{"exec", "--container-id", containerID, "--workspace-folder", workspaceFolder,
			"--override-config", configFilePath, containerPortForwarderPath, "-l", "0.0.0.0:0", "-f", target}
		if err := s.startPortForwarder(containerIp, fc.Port, target, devcontainerPath, args); err != nil {
			return err
		}
	}
	return nil
}

func (s *portForwardingSession) startPortForwarder(containerIP, port, target, command string, args []string) error {
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
	if err := portforward.ServeForwarding(ctx, listener, destination); err != nil {
		fmt.Fprintf(os.Stderr, "port forwarding on %s failed: %v\n", addr, err)
	}
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

	ctx, cancel := context.WithCancel(ctx)
	session := &portForwardingSession{ctx: ctx}
	stop := func() {
		cancel()
		session.workers.Wait()
	}
	// Start a child for each configured target so the listening port comes
	// from a forwarder owned by this session.
	if err := session.startPortForwarders(containerID, containerIp, devcontainerPath, workspaceFolder, configFilePath, forwardConfigs); err != nil {
		stop()
		return nil, err
	}
	return stop, nil
}
