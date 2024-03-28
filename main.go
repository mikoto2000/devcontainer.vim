// TODO: vimFilePath の名前変更(hostVimFilePath?)

package main

import (
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/dockerRun"
	"github.com/mikoto2000/devcontainer.vim/util"
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

const APP_NAME = "devcontainer.vim"
const VIM_TAG_NAME = "v9.1.0181"
const VIM_DOWNLOAD_URL = "https://github.com/vim/vim-appimage/releases/download/%s/"
const VIM_FILE_NAME = "Vim-%s.glibc2.29-x86_64.AppImage"

func main() {
	// コマンドラインオプションのパース

	// devcontainer.vim 用のディレクトリ作成
	// 1. ユーザーコンフィグ用ディレクトリ
	//    `os.UserConfigDir` + `devcontainer.vim`
	// 2. ユーザーキャッシュ用ディレクトリ
	//    `os.UserCacheDir` + `devcontainer.vim`
	util.CreateDirectory(os.UserConfigDir, APP_NAME)
	appCacheDir := util.CreateDirectory(os.UserCacheDir, APP_NAME)

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
					isExistsDocker := util.IsExistsCommand("docker")
					if !isExistsDocker {
						fmt.Fprintf(os.Stderr, "docker not found.")
						os.Exit(1)
					}

					// コンテナ起動
					dockerRun.ExecuteDockerRun(cCtx.Args().Slice(), vimFilePath, vimFileName)

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
	if util.IsExists(vimFilePath) {
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
