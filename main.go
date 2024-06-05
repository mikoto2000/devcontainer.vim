package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/docker"
	"github.com/mikoto2000/devcontainer.vim/tools"
	"github.com/mikoto2000/devcontainer.vim/util"
)

var version string

const FLAG_NAME_LICENSE = "license"
const FLAG_NAME_HELP_LONG = "help"
const FLAG_NAME_HELP_SHORT = "h"
const FLAG_NAME_VERSION_LONG = "version"
const SPLIT_ARG_MARK = "--"

const FLAG_NAME_GENERATE = "generate"
const FLAG_NAME_HOME = "home"
const FLAG_NAME_OUTPUT = "output"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

//go:embed devcontainer.vim.template.json
var devcontainerVimJsonTemplate string

const APP_NAME = "devcontainer.vim"

func main() {
	// Windows でも `${ localEnv:HOME }` でホームディレクトリの指定ができるように、
	// 環境変数を更新
	if runtime.GOOS == "windows" {
		fmt.Printf("Set environment variable HOME to %s.\n", os.Getenv("USERPROFILE"))
		os.Setenv("HOME", os.Getenv("USERPROFILE"))
	}

	// コマンドラインオプションのパース

	// devcontainer.vim 用のディレクトリ作成
	// 1. ユーザーコンフィグ用ディレクトリ
	//    `os.UserConfigDir` + `devcontainer.vim`
	// 2. ユーザーキャッシュ用ディレクトリ
	//    `os.UserCacheDir` + `devcontainer.vim`
	appCacheDir, binDir, configDirForDocker, configDirForDevcontainer := util.CreateDirectory(os.UserCacheDir, APP_NAME)

	devcontainerVimArgProcess := (&cli.App{
		Name:                   "devcontainer.vim",
		Usage:                  "devcontainer for vim.",
		Version:                version,
		UseShortOptionHandling: true,
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

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:            "run",
				Usage:           "Run container use `docker run`",
				UsageText:       "devcontainer.vim run [DOCKER_OPTIONS...] [DOCKER_ARGS...]",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// `docker run` でコンテナを立てる

					// Requirements のチェック
					// 1. docker
					isExistsDocker := util.IsExistsCommand("docker")
					if !isExistsDocker {
						fmt.Fprintf(os.Stderr, "docker not found.")
						os.Exit(1)
					}

					// 必要なファイルのダウンロード
					vimPath, cdrPath, err := tools.InstallRunTools(binDir)
					if err != nil {
						panic(err)
					}

					// clipboard-data-receiver を起動
					pid, port, err := tools.RunCdr(cdrPath, configDirForDocker)
					if err != nil {
						panic(err)
					}
					fmt.Printf("Started clipboard-data-receiver with pid: %d, port: %d\n", pid, port)

					// コンテナ起動
					docker.Run(cCtx.Args().Slice(), vimPath, cdrPath, configDirForDocker)

					// clipboard-data-receiver を停止
					process, err := os.FindProcess(pid)
					if err != nil {
						panic(err)
					}
					process.Kill()

					return nil
				},
			},
			{
				Name:            "templates",
				Usage:           "Run `devcontainer templates`",
				UsageText:       "devcontainer.vim templates [DEVCONTAINER_OPTIONS...] WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// devcontainer の template サブコマンド実行

					// 必要なファイルのダウンロード
					devcontainerFilePath, err := tools.InstallTemplatesTools(binDir)
					if err != nil {
						panic(err)
					}

					// devcontainer を用いたコンテナ立ち上げ
					output, _ := devcontainer.Templates(devcontainerFilePath, cCtx.Args().Slice()...)
					fmt.Println(output)

					return nil
				},
			},
			{
				Name:            "start",
				Usage:           "Run `devcontainer up` and `devcontainer exec`",
				UsageText:       "devcontainer.vim start [DEVCONTAINER_OPTIONS...] WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// devcontainer でコンテナを立てる

					// 必要なファイルのダウンロード
					vimPath, devcontainerFilePath, err := tools.InstallStartTools(binDir)
					if err != nil {
						panic(err)
					}

					// コマンドライン引数の末尾は `--workspace-folder` の値として使う
					args := cCtx.Args().Slice()
					workspaceFolder := args[len(args)-1]
					configFilePath, err := createConfigFile(devcontainerFilePath, workspaceFolder, configDirForDevcontainer)
					if err != nil {
						panic(err)
					}

					// devcontainer を用いたコンテナ立ち上げ
					devcontainer.ExecuteDevcontainer(args, devcontainerFilePath, vimPath, configFilePath)

					return nil
				},
			},
			{
				Name:            "down",
				Usage:           "Stop and remove devcontainers.",
				UsageText:       "devcontainer.vim down WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// devcontainer でコンテナを立てる

					// 必要なファイルのダウンロード
					devcontainerPath, err := tools.InstallDownTools(binDir)
					if err != nil {
						panic(err)
					}

					// devcontainer を用いたコンテナ終了
					devcontainer.Down(cCtx.Args().Slice(), devcontainerPath)

					// 設定ファイルを削除
					// コマンドライン引数の末尾は `--workspace-folder` の値として使う
					args := cCtx.Args().Slice()
					workspaceFolder := args[len(args)-1]
					configDir := util.GetConfigDir(appCacheDir, workspaceFolder)

					fmt.Printf("Remove configuration file: `%s`\n", configDir)
					os.RemoveAll(configDir)

					return nil
				},
			},
			{
				Name:            "config",
				Usage:           "devcontainer.vim's config information.",
				UsageText:       "devcontainer.vim config [OPTIONS...]",
				HideHelp:        false,
				SkipFlagParsing: false,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    FLAG_NAME_GENERATE,
						Aliases: []string{"g"},
						Value:   false,
						Usage:   "generate sample config file.",
					},
					&cli.StringFlag{
						Name:    FLAG_NAME_HOME,
						Aliases: []string{},
						Value:   "/home/vscode",
						Usage:   "generate sample config's home directory.",
					},
					&cli.StringFlag{
						Name:    FLAG_NAME_OUTPUT,
						Aliases: []string{"o"},
						Value:   ".devcontainer/devcontainer.vim.json",
						Usage:   "generate sample config output file path.",
					},
				},
				Action: func(cCtx *cli.Context) error {
					// 何かしらオプションでない引数を渡されたらヘルプを出力して終了
					if cCtx.NumFlags() == 0 || cCtx.Args().Present() {
						cli.ShowSubcommandHelpAndExit(cCtx, 0)
					}

					// generate フラグがセットされていたら設定ファイルのひな形を出力する
					if cCtx.Bool(FLAG_NAME_GENERATE) {

						// home オプションで指定された値を利用して、バインド先を置換
						devcontainerVimJson := strings.Replace(devcontainerVimJsonTemplate, "{{ remoteEnv:HOME }}", cCtx.String(FLAG_NAME_HOME), -1)

						if cCtx.IsSet(FLAG_NAME_OUTPUT) {
							// output オプションが指定されている場合、指定されたパスへ出力する
							configFilePath := cCtx.String(FLAG_NAME_OUTPUT)

							// 生成先ディレクトリを作成
							err := os.MkdirAll(filepath.Dir(configFilePath), 0766)
							if err != nil {
								panic(err)
							}

							// 設定ファイルサンプルを出力
							err = os.WriteFile(configFilePath, []byte(devcontainerVimJson), 0666)
							if err != nil {
								panic(err)
							}
						} else {
							// output オプションが指定されていない場合、標準出力へ出力する
							fmt.Print(devcontainerVimJson)
						}
					}

					return nil
				},
			},
			{
				Name:            "tool",
				Usage:           "Management tools",
				UsageText:       "devcontainer.vim tool SUB_COMMAND",
				HideHelp:        false,
				SkipFlagParsing: false,
				Subcommands: []*cli.Command{
					{
						Name:            "vim",
						Usage:           "Management vim",
						UsageText:       "devcontainer.vim tool vim SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly vim",
								UsageText:       "devcontainer.vim tool vim download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Action: func(cCtx *cli.Context) error {

									// Vim のダウンロード
									_, err := tools.VIM.Install(binDir, true)
									if err != nil {
										panic(err)
									}

									return nil
								},
							},
						},
					},
					{
						Name:            "devcontainer",
						Usage:           "Management devcontainer cli",
						UsageText:       "devcontainer.vim tool devcontainer SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly devcontainer cli",
								UsageText:       "devcontainer.vim tool devcontainer download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Action: func(cCtx *cli.Context) error {

									// devcontainer のダウンロード
									_, err := tools.DEVCONTAINER.Install(binDir, true)
									if err != nil {
										panic(err)
									}

									return nil
								},
							},
						},
					},
					{
						Name:            "clipboard-data-receiver",
						Usage:           "Management clipboard-data-receiver",
						UsageText:       "devcontainer.vim tool clipboard-data-receiver SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly devcontainer cli",
								UsageText:       "devcontainer.vim tool devcontainer download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Action: func(cCtx *cli.Context) error {

									// clipboard-data-receiver のダウンロード
									_, err := tools.CDR.Install(binDir, true)
									if err != nil {
										panic(err)
									}

									return nil
								},
							},
						},
					},
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

// devcontainer.vim 起動時に使用する設定ファイルを作成する
// 設定ファイルは、 devcontainer.vim のキャッシュ内の `config` ディレクトリに、
// ワークスペースフォルダのパスを md5 ハッシュ化した名前のディレクトリに格納する.
func createConfigFile(devcontainerFilePath string, workspaceFolder string, configDirForDevcontainer string) (string, error) {
	// devcontainer の設定ファイルパス取得
	configFilePath, err := devcontainer.GetConfigurationFilePath(devcontainerFilePath, workspaceFolder)
	if err != nil {
		return "", err
	}

	// devcontainer.vim 用の追加設定ファイルを探す
	configurationFileName := configFilePath[:len(configFilePath)-len(filepath.Ext(configFilePath))]
	additionalConfigurationFilePath := configurationFileName + ".vim.json"

	// 設定管理フォルダに JSON を配置
	mergedConfigFilePath, err := util.CreateConfigFileForDevcontainer(configDirForDevcontainer, workspaceFolder, configFilePath, additionalConfigurationFilePath)

	fmt.Printf("Use configuration file: `%s`", mergedConfigFilePath)

	return mergedConfigFilePath, err
}
