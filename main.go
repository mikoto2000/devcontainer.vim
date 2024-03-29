// TODO: vimFilePath の名前変更(hostVimFilePath?)

package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

const FLAG_NAME_LICENSE = "license"
const FLAG_NAME_HELP_LONG = "help"
const FLAG_NAME_HELP_SHORT = "h"
const FLAG_NAME_VERSION_LONG = "version"
const SPLIT_ARG_MARK = "--"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

const CONTAINER_COMMAND = "docker"

var DOCKER_RUN_ARGS_PREFIX = []string{"run", "-d", "--rm"}
var DOCKER_RUN_ARGS_SUFFIX = []string{"sh", "-c", "trap \"exit 0\" TERM; sleep infinity & wait"}

const APP_NAME = "devcontainer.vim"
const VIM_TAG_NAME = "v9.1.0181"
const VIM_DOWNLOAD_URL = "https://github.com/vim/vim-appimage/releases/download/%s/"
const VIM_FILE_NAME = "Vim-%s.glibc2.29-x86_64.AppImage"

type GetDirFunc func() (string, error)

func main() {
	// コマンドラインオプションのパース

	// devcontainer.vim 用のディレクトリ作成
	// 1. ユーザーコンフィグ用ディレクトリ
	//    `os.UserConfigDir` + `devcontainer.vim`
	// 2. ユーザーキャッシュ用ディレクトリ
	//    `os.UserCacheDir` + `devcontainer.vim`
	createDirectory(os.UserConfigDir, APP_NAME)
	appCacheDir := createDirectory(os.UserCacheDir, APP_NAME)

	// Vim 関連の文字列組み立て
	vimFileName := fmt.Sprintf(VIM_FILE_NAME, VIM_TAG_NAME)
	vimFilePath := filepath.Join(appCacheDir, vimFileName)

	devcontainerVimArgProcess := (&cli.App{
		Name:                   "devcontainer.vim",
		Usage:                  "devcontainer for vim.",
		Version:                "0.0.1",
		UseShortOptionHandling: true,
		HideHelpCommand:        true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:               FLAG_NAME_LICENSE,
				Aliases:            []string{"l"},
				Value:              false,
				DisableDefaultText: true,
				Usage:              "show licensesa.",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// ライセンスフラグが立っていればライセンスを表示して終
			if cCtx.Bool(FLAG_NAME_LICENSE) {
				fmt.Println(license)
				fmt.Println()
				fmt.Println(notice)
				os.Exit(0)
			}

			// TODO: フラグをパースして後続に渡すための変数へ格納していく
			return nil
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:            "run",
				Usage:           "Run container use `docker run`",
				UsageText:       "devcontainer.vim run [DOCKER_OPTIONS...] [DOCKER_ARGS...]",
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// 必要なファイルのダウンロード
					downloadFiles(appCacheDir, vimFilePath, vimFileName)

					// Requirements のチェック
					// 1. docker
					isExistsDocker := isExistsCommand("docker")
					if !isExistsDocker {
						fmt.Fprintf(os.Stderr, "docker not found.")
						os.Exit(1)
					}

					// コンテナ起動
					startDevContainer(cCtx.Args().Slice(), vimFilePath, vimFileName)

					return nil
				},
			},
		},
	})

	// アプリ実行
	err := devcontainerVimArgProcess.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}

func downloadFiles(appCacheDir string, vimFilePath string, vimFileName string) {
	// vim-appimage のダウンロード
	// 1. ユーザーキャッシュディレクトリ取得
	// 2. appimage がダウンロード済みかをチェックし、
	//    必要であればダウンロード
	if isExistsVimAppImage(vimFilePath) {
		fmt.Printf("Vim AppImage aleady exist, use %s.\n", vimFilePath)
	} else {
		vimDownloadUrl := fmt.Sprintf(VIM_DOWNLOAD_URL+vimFileName, VIM_TAG_NAME)
		fmt.Printf("Download Vim AppImage from %s ...", vimDownloadUrl)
		err := downloadVimAppImage(vimDownloadUrl, appCacheDir, vimFileName)
		if err != nil {
			panic(err)
		}
		fmt.Printf(" done.\n")
	}
}

func startDevContainer(args []string, vimFilePath string, vimFileName string) {
	// バックグラウンドでコンテナを起動
	// `docker run -d --rm os.Args[1:] sh -c "sleep infinity"`
	dockerRunArgs := append(DOCKER_RUN_ARGS_PREFIX, args...)
	dockerRunArgs = append(dockerRunArgs, DOCKER_RUN_ARGS_SUFFIX...)
	fmt.Printf("run container: `%s \"%s\"`\n", CONTAINER_COMMAND, strings.Join(dockerRunArgs, "\" \""))
	dockerRunCommand := exec.Command(CONTAINER_COMMAND, dockerRunArgs...)
	containerIdRaw, err := dockerRunCommand.CombinedOutput()
	containerId := string(containerIdRaw)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Container start error.")
		fmt.Fprintln(os.Stderr, string(containerId))
		panic(err)
	}
	containerId = strings.ReplaceAll(containerId, "\n", "")
	containerId = strings.ReplaceAll(containerId, "\r", "")
	fmt.Printf("Container started. id: %s\n", containerId)

	// コンテナへ appimage を転送して実行権限を追加
	// `docker cp <os.UserCacheDir/devcontainer.vim/Vim-AppImage> <dockerrun 時に標準出力に表示される CONTAINER ID>:/`
	dockerCpArgs := []string{"cp", vimFilePath, containerId + ":/"}
	fmt.Printf("Copy AppImage: `%s \"%s\"` ...", CONTAINER_COMMAND, strings.Join(dockerCpArgs, "\" \""))
	copyResult, err := exec.Command(CONTAINER_COMMAND, dockerCpArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "AppImage copy error.")
		fmt.Fprintln(os.Stderr, string(copyResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> chmod +x /Vim-AppImage`
	dockerChownArgs := []string{"exec", containerId, "sh", "-c", "chmod +x /" + vimFileName}
	fmt.Printf("Chown AppImage: `%s \"%s\"` ...", CONTAINER_COMMAND, strings.Join(dockerChownArgs, "\" \""))
	chmodResult, err := exec.Command(CONTAINER_COMMAND, dockerChownArgs...).CombinedOutput()
	if err != nil {
		fmt.Fprintln(os.Stderr, "chmod error.")
		fmt.Fprintln(os.Stderr, string(chmodResult))
		panic(err)
	}
	fmt.Printf(" done.\n")

	// コンテナへ接続
	// `docker exec <dockerrun 時に標準出力に表示される CONTAINER ID> /Vim-AppImage`

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dockerVimArgs := []string{"exec", "-it", containerId, "/" + vimFileName, "--appimage-extract-and-run"}
	fmt.Printf("Start vim: `%s \"%s\"`", CONTAINER_COMMAND, strings.Join(dockerVimArgs, "\" \""))
	dockerExec := exec.CommandContext(ctx, CONTAINER_COMMAND, dockerVimArgs...)
	dockerExec.Stdin = os.Stdin
	dockerExec.Stdout = os.Stdout
	dockerExec.Stderr = os.Stderr
	dockerExec.Cancel = func() error {
		fmt.Fprintf(os.Stderr, "Receive SIGINT.\n")
		return dockerExec.Process.Signal(os.Interrupt)
	}

	err = dockerExec.Run()
	if err != nil {
		panic(err)
	}

	// コンテナ停止
	// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
	fmt.Printf("Stop container(Async) %s.\n", containerId)
	err = exec.Command(CONTAINER_COMMAND, "stop", containerId).Start()
	if err != nil {
		panic(err)
	}
}

func isExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		return false
	}
	return true
}

func createDirectory(pathFunc GetDirFunc, dirName string) string {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appCacheDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, 0766); err != nil {
		panic(err)
	}
	return appCacheDir
}

func isExistsVimAppImage(vimFilePath string) bool {
	_, err := os.Stat(vimFilePath)
	return err == nil
}

func downloadVimAppImage(vimDownloadUrl string, appCacheDir string, vimFileName string) error {
	vimFilePath := filepath.Join(appCacheDir, vimFileName)

	// HTTP GETリクエストを送信
	resp, err := http.Get(vimDownloadUrl)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ファイルを作成
	out, err := os.Create(vimFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// レスポンスの内容をファイルに書き込み
	_, err = io.Copy(out, resp.Body)
	return err
}
