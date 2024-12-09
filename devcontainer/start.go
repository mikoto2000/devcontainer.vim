package devcontainer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

var devcontainreArgsPrefix = []string{"up"}

// devcontainer でコンテナを立ち上げ、 Vim を転送し、実行する。
// 既存実装の都合上、configFilePath から configDirForDevcontainer を抽出している
func Start(
	args []string,
	devcontainerPath string,
	cdrPath string,
	vimInstallDir string,
	nvim bool,
	configFilePath string,
	vimrc string) error {

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	workspaceFolder := args[len(args)-1]

	// `devcontainer up` でコンテナを起動

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
		return err
	}

	upCommandResult, err := UnmarshalUpCommandResult(stdout)
	if err != nil {
		return err
	}

	containerID := upCommandResult.ContainerID

	fmt.Printf("finished devcontainer up: %s\n", upCommandResult)

	// コンテナ内に入り、コンテナの Arch を確認
	containerArch, err := docker.Exec(containerID, "uname", "-m")
	if err != nil {
		return err
	}
	containerArch = strings.TrimSpace(containerArch)
	containerArch, err = util.NormalizeContainerArch(containerArch)
	if err != nil {
		return err
	}
	fmt.Printf("Container Arch: '%s'.\n", containerArch)

	portForwarderContainerPath, err := tools.PortForwarderContainer.Install(vimInstallDir, containerArch, false)
	if err != nil {
		return err
	}
	err = docker.Cp("port-forwarder-container", portForwarderContainerPath, containerID, "/port-forwarder")
	if err != nil {
		return err
	}

	// clipboard-data-receiver を起動
	configDirForDevcontainer := filepath.Dir(configFilePath)
	pid, port, err := tools.RunCdr(cdrPath, configDirForDevcontainer)
	if err != nil {
		return err
	}
	fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

	// コンテナの IP アドレスを取得
	containerIp, err := docker.Exec(containerID, "sh", "-c", "hostname -i")
	if err != nil {
		return err
	}
	containerIp = strings.TrimSpace(containerIp)

	// すでに port-forwarder が起動しているなら実行しない
	psOut, err := docker.Exec(containerID, "sh", "-c", "grep --files-with-matches port-forwarder /proc/*/comm || true")
	if err != nil {
		return err
	}
	fmt.Printf("Running port-forwarders: %s\n", strings.Split(strings.TrimSpace(psOut), "\n"))

	pfCount := strings.Split(strings.TrimSpace(psOut), "\n")
	pfCount = util.RemoveEmptyString(pfCount)

	if len(pfCount) == 0 {
		fmt.Println("Start port-forwarder in container.")

		// forwardPorts を解釈してport-forwarder を実行

		// forwardPorts を解釈
		configurationString, err := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder)
		if err != nil {
			return err
		}
		forwardConfigs, err := GetForwardPorts(configurationString)

		// 解釈した forwardPort ごとに port-forwarder を起動する
		for _, fc := range forwardConfigs {

			// コンテナ側の port-forwarder の起動
			portForwarderCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
			defer cancel()
			fmt.Printf("%s %s %s %s %s %s %s.\n", devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "/port-forwarder -l 0.0.0.0:0 -f "+fc.Host+":"+fc.Port)
			dockerExecPortForwarder := exec.CommandContext(portForwarderCtx, devcontainerPath, "exec", "--workspace-folder", ".", "sh", "-c", "/port-forwarder -l 0.0.0.0:0 -f "+fc.Host+":"+fc.Port)
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
				reader := bufio.NewReader(portOut)
				for {
					port, err := reader.ReadString('\n')
					if err != nil {
						if err != io.EOF {
							fmt.Println("Error reading from stdout:", err)
						}
						break
					}
					port = strings.TrimSpace(port)
					fmt.Printf("port-forwarder started: %s:%s %s\n", containerIp, port, fc.Host+":"+fc.Port)

					// forwardPorts の内容を `/pf` ディテク取りに「<転送先>_<リッスンアドレス＆ポート>」の形式で配置する
					docker.Exec(containerID, "sh", "-c", "mkdir -p /pf && touch /pf/"+fc.Host+":"+fc.Port+"_"+containerIp+":"+port)

					util.StartForwarding("0.0.0.0:"+fc.Port, containerIp+":"+port)
				}

			}()
		}
	} else {
		fmt.Println("port-forwarder already running.")

		// `/pf` ディレクトリの内容からフォワードするポートを解釈し、フォワードする
		lspfOut, err := docker.Exec(containerID, "sh", "-c", "ls --zero /pf")
		if err != nil {
			return err
		}
		forwardConfigs := strings.Split(lspfOut, "\x00")
		for _, forwardConfig := range forwardConfigs {
			if len(forwardConfig) == 0 {
				continue
			}
			splitedForwardConfig := strings.Split(forwardConfig, "_")
			containerSrc := splitedForwardConfig[0]
			scs := strings.Split(containerSrc, ":")
			containerSrcPort := scs[1]
			containerDest := splitedForwardConfig[1]
			scd := strings.Split(containerDest, ":")
			containerDestPort := scd[1]

			go func() {
				fmt.Printf("listen: %s, forward: %s.\n", "0.0.0.0:"+containerSrcPort, containerIp+":"+containerDestPort)
				util.StartForwarding("0.0.0.0:"+containerSrcPort, containerIp+":"+containerDestPort)
			}()
		}
	}

	vimFilePath, err := tools.InstallVim(vimInstallDir, nvim, containerArch)
	if err != nil {
		return err
	}
	// vim_<ARCH>, nvim_<ARCH> の形式でパスがわたってくるので、
	// vim/nvim の部分を抽出する。
	vimFileName := strings.Split(filepath.Base(vimFilePath), "_")[0]

	useSystemVim := false
	fmt.Printf("Check system installed %s ... ", vimFileName)
	out, _ := docker.Exec(containerID, "which", vimFileName)
	if out != "" {
		fmt.Printf("found.\n")
		useSystemVim = true
	} else {
		fmt.Printf("not found.\n")

		if runtime.GOARCH == "arm64" {
			// arm の場合スタティックリンクの nvim を作れないため、 vim にフォールバック
			vimFileName = "vim"
			nvim = false
		} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" && nvim {
			// M1 Mac で amd64 のコンテナを動かすと、なぜか AppImage が動かないので vim にフォールバック
			vimFileName = "vim"
			nvim = false
		}
	}
	fmt.Printf("docker exec output: \"%s\".\n", strings.TrimSpace(out))

	if !useSystemVim {
		// コンテナへ appimage を転送して実行権限を追加
		// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
		err = docker.Cp("vim", vimFilePath, containerID, "/"+vimFileName)
		if err != nil {
			return err
		}

		// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
		dockerChownArgs := []string{"exec", "--user", "root", containerID, "sh", "-c", "chmod +x /" + vimFileName}
		fmt.Printf("Chown AppImage: `%s \"%s\"` ...", containerCommand, strings.Join(dockerChownArgs, "\" \""))
		chmodResult, err := exec.Command(containerCommand, dockerChownArgs...).CombinedOutput()
		if err != nil {
			fmt.Fprintln(os.Stderr, "chmod error.")
			fmt.Fprintln(os.Stderr, string(chmodResult))
			return err
		}
		fmt.Printf(" done.\n")
	}

	// Vim 関連ファイルの転送(`SendToTcp.vim` と、追加の `vimrc`)
	sendToTCP, err := tools.CreateSendToTCP(configDirForDevcontainer, port, vimFileName == "nvim")
	if err != nil {
		return err
	}

	// コンテナへ SendToTcp.vim を転送
	err = docker.Cp("SendToTcp.vim", sendToTCP, containerID, "/")
	if err != nil {
		return err
	}

	// コンテナへ vimrc を転送
	err = docker.Cp("vimrc", vimrc, containerID, "/")
	if err != nil {
		return err
	}

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	sendToTCPName := filepath.Base(sendToTCP)
	devcontainerStartVimArgs := devcontainerStartVimArgs(containerID, workspaceFolder, vimFileName, sendToTCPName, containerArch, useSystemVim)
	fmt.Printf("Start vim: `%s \"%s\"`\n", devcontainerPath, strings.Join(devcontainerStartVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, devcontainerPath, devcontainerStartVimArgs...)
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
	}

	// コンテナ停止は別途 down コマンドで行う
	return nil
}
