// TODO: vimFilePath の名前変更(hostVimFilePath?)

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/devcontainreUpAndExec"
	"github.com/mikoto2000/devcontainer.vim/dockerRun"
	"github.com/mikoto2000/devcontainer.vim/tools"
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

func main() {
	// コマンドラインオプションのパース

	// devcontainer.vim 用のディレクトリ作成
	// 1. ユーザーコンフィグ用ディレクトリ
	//    `os.UserConfigDir` + `devcontainer.vim`
	// 2. ユーザーキャッシュ用ディレクトリ
	//    `os.UserCacheDir` + `devcontainer.vim`
	util.CreateDirectory(os.UserConfigDir, APP_NAME)
	appCacheDir := util.CreateDirectory(os.UserCacheDir, APP_NAME)

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

			// devcontainer でコンテナを立てる

			// 必要なファイルのダウンロード
			vim, err := tools.VIM.Install(appCacheDir)
			if err != nil {
				panic(err)
			}

			devcontainer, err := tools.DEVCONTAINER.Install(appCacheDir)
			if err != nil {
				panic(err)
			}

			// TODO: devcontainer を用いた処理を実装
			// `devcontainer up` でコンテナ起動
			// vim をコンテナへコピー
			// `devcontainer exec` でコンテナの vim を起動
			devcontainreUpAndExec.ExecuteDevcontainer([]string{}, devcontainer, vim)

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:            "run",
				Usage:           "Run container use `docker run`",
				UsageText:       "devcontainer.vim run [DOCKER_OPTIONS...] [DOCKER_ARGS...]",
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// `docker run` でコンテナを立てる

					// 必要なファイルのダウンロード
					vim, err := tools.VIM.Install(appCacheDir)
					if err != nil {
						panic(err)
					}

					// Requirements のチェック
					// 1. docker
					isExistsDocker := util.IsExistsCommand("docker")
					if !isExistsDocker {
						fmt.Fprintf(os.Stderr, "docker not found.")
						os.Exit(1)
					}

					// コンテナ起動
					dockerRun.ExecuteDockerRun(cCtx.Args().Slice(), vim)

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
