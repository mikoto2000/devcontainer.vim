package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/docker"
	"github.com/mikoto2000/devcontainer.vim/oras"
	"github.com/mikoto2000/devcontainer.vim/tools"
	"github.com/mikoto2000/devcontainer.vim/util"
)

type IndexRoot struct {
	Collections []Collection `json:"collections"`
}

type Collection struct {
	Templates []AvailableTemplateItem `json:"templates"`
}

type AvailableTemplateItem struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

var version string

const flagNameLicense = "license"

const flagNameGenerate = "generate"
const flagNameHome = "home"
const flagNameOutput = "output"
const flagNameOpen = "open"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

//go:embed devcontainer.vim.template.json
var devcontainerVimJSONTemplate string

const runargsContent = "-v \"$(pwd):/work\" -v \"$HOME/.vim:/root/.vim\" -v \"$HOME/.gitconfig:/root/.gitconfig\" -v \"$HOME/.ssh:/root/.ssh\" --workdir /work"

//go:embed vimrc.template.vim
var additionalVimrc string

const appName = "devcontainer.vim"

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
	appConfigDir := util.CreateConfigDirectory(os.UserConfigDir, appName)
	appCacheDir, binDir, configDirForDocker, configDirForDevcontainer := util.CreateCacheDirectory(os.UserCacheDir, appName)

	// vimrc ファイルの出力先を組み立て
	// vimrc を出力(既に存在するなら何もしない)
	vimrc := filepath.Join(appConfigDir, "vimrc")
	if !util.IsExists(vimrc) {
		err := util.CreateFileWithContents(vimrc, additionalVimrc, 0666)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Generated additional vimrc to: %s\n", vimrc)
	}

	// runargs ファイルの出力先を組み立て
	// runargs を出力(既に存在するなら何もしない)
	runargs := filepath.Join(appConfigDir, "runargs")
	if !util.IsExists(runargs) {
		err := util.CreateFileWithContents(runargs, runargsContent, 0666)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Generated additional runargs to: %s\n", runargs)
	}

	devcontainerVimArgProcess := (&cli.App{
		Name:                   "devcontainer.vim",
		Usage:                  "devcontainer for vim.",
		Version:                version,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:               flagNameLicense,
				Aliases:            []string{"l"},
				Value:              false,
				DisableDefaultText: true,
				Usage:              "show licensesa.",
			},
		},
		Action: func(cCtx *cli.Context) error {
			// ライセンスフラグが立っていればライセンスを表示して終
			if cCtx.Bool(flagNameLicense) {
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

					// デフォルト引数取得
					defaultRunargsBytes, err := os.ReadFile(runargs)
					if err != nil {
						panic(err)
					}
					defaultRunargsString := string(defaultRunargsBytes)

					if runtime.GOOS == "windows" {
						// コンテナ起動
						// windows はシェル変数展開が上手くいかないので runargs を使用しない
						docker.Run(cCtx.Args().Slice(), vimPath, cdrPath, configDirForDocker, vimrc, []string{})
					} else {
						// デフォルト引数内のシェル変数を展開
						extractedDofaultRunargsString, err := util.ExtractShellVariables(defaultRunargsString)
						if err != nil {
							panic(err)
						}

						// 展開したものを配列へ分割
						defaultRunargs, err := shlex.Split(extractedDofaultRunargsString, true)
						if err != nil {
							panic(err)
						}

						// コンテナ起動
						docker.Run(cCtx.Args().Slice(), vimPath, cdrPath, configDirForDocker, vimrc, defaultRunargs)
					}

					return nil
				},
			},
			{
				Name:      "templates",
				Usage:     "Run `devcontainer templates`",
				UsageText: "devcontainer.vim templates [DEVCONTAINER_OPTIONS...] WORKSPACE_FOLDER",
				Subcommands: []*cli.Command{
					{
						Name:      "apply",
						Usage:     "Apply template.",
						UsageText: "devcontainer.vim templates apply WORKSPACE_FOLDER",
						Action: func(cCtx *cli.Context) error {

							// Features の一覧をダウンロード
							indexFileName := "devcontainer-index.json"
							indexFile := filepath.Join(appCacheDir, indexFileName)
							if !util.IsExists(indexFile) {
								fmt.Println("Download template index ... ")
								oras.Pull("ghcr.io/devcontainers/index", "latest", appCacheDir)
								fmt.Println("done.")
							}

							var indexRoot IndexRoot
							jsonFile := filepath.Join(appCacheDir, indexFileName)
							jsonData, err := os.ReadFile(jsonFile)
							err = json.Unmarshal([]byte(jsonData), &indexRoot)
							if err != nil {
								panic(err)
							}

							var availableTemplateItems []AvailableTemplateItem
							for _, collection := range indexRoot.Collections {
								availableTemplateItems = append(availableTemplateItems, collection.Templates...)
							}

							names := []string{}
							for _, item := range availableTemplateItems {
								names = append(names, item.Name)
							}

							prompt := promptui.Select{
								Label:             "Select Template",
								Items:             names,
								StartInSearchMode: true,
								Searcher: func(input string, index int) bool {
									item := names[index]
									name := strings.Replace(strings.ToLower(item), " ", "", -1)
									input = strings.Replace(strings.ToLower(input), " ", "", -1)

									return strings.Contains(name, input)
								},
							}

							i, _, err := prompt.Run()
							if err != nil {
								panic(err)
							}

							selectedItem := availableTemplateItems[i]

							// devcontainer の template サブコマンド実行

							// 必要なファイルのダウンロード
							devcontainerFilePath, err := tools.InstallTemplatesTools(binDir)
							if err != nil {
								panic(err)
							}

							// コマンドライン引数の末尾は `--workspace-folder` の値として使う
							args := cCtx.Args().Slice()
							workspaceFolder := args[len(args)-1]

							templateID := selectedItem.ID + ":" + selectedItem.Version

							// devcontainer を用いたコンテナ立ち上げ
							output, _ := devcontainer.Templates(
								devcontainerFilePath,
								workspaceFolder,
								templateID)

							fmt.Println(output)

							return nil
						},
					},
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
					vimPath, devcontainerPath, cdrPath, err := tools.InstallStartTools(binDir)
					if err != nil {
						panic(err)
					}

					// コマンドライン引数の末尾は `--workspace-folder` の値として使う
					args := cCtx.Args().Slice()
					workspaceFolder := args[len(args)-1]
					configFilePath, err := createConfigFile(devcontainerPath, workspaceFolder, configDirForDevcontainer)
					if err != nil {
						panic(err)
					}

					// devcontainer を用いたコンテナ立ち上げ
					devcontainer.ExecuteDevcontainer(args, devcontainerPath, vimPath, cdrPath, configFilePath, vimrc)

					return nil
				},
			},
			{
				Name:            "stop",
				Usage:           "Stop devcontainers.",
				UsageText:       "devcontainer.vim stop WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action: func(cCtx *cli.Context) error {
					// devcontainer でコンテナを立てる

					// 必要なファイルのダウンロード
					devcontainerPath, err := tools.InstallStopTools(binDir)
					if err != nil {
						panic(err)
					}

					// devcontainer を用いたコンテナ終了
					devcontainer.Stop(cCtx.Args().Slice(), devcontainerPath, configDirForDevcontainer)

					fmt.Printf("Stop containers\n")

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
					devcontainer.Down(cCtx.Args().Slice(), devcontainerPath, configDirForDevcontainer)

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
						Name:    flagNameGenerate,
						Aliases: []string{"g"},
						Value:   false,
						Usage:   "generate sample config file.",
					},
					&cli.StringFlag{
						Name:    flagNameHome,
						Aliases: []string{},
						Value:   "/home/vscode",
						Usage:   "generate sample config's home directory.",
					},
					&cli.StringFlag{
						Name:    flagNameOutput,
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
					if cCtx.Bool(flagNameGenerate) {

						// home オプションで指定された値を利用して、バインド先を置換
						devcontainerVimJSON := strings.Replace(devcontainerVimJSONTemplate, "{{ remoteEnv:HOME }}", cCtx.String(flagNameHome), -1)

						if cCtx.IsSet(flagNameOutput) {
							// output オプションが指定されている場合、指定されたパスへ出力する
							configFilePath := cCtx.String(flagNameOutput)

							// 生成先ディレクトリを作成
							err := os.MkdirAll(filepath.Dir(configFilePath), 0766)
							if err != nil {
								panic(err)
							}

							// 設定ファイルサンプルを出力
							err = os.WriteFile(configFilePath, []byte(devcontainerVimJSON), 0666)
							if err != nil {
								panic(err)
							}
						} else {
							// output オプションが指定されていない場合、標準出力へ出力する
							fmt.Print(devcontainerVimJSON)
						}
					}

					return nil
				},
			},
			{
				Name:            "vimrc",
				Usage:           "devcontainer.vim's vimrc information.",
				UsageText:       "devcontainer.vim vimrc [OPTIONS...]",
				HideHelp:        false,
				SkipFlagParsing: false,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    flagNameGenerate,
						Aliases: []string{"g"},
						Value:   false,
						Usage:   "regenerate vimrc file.",
					},
					&cli.BoolFlag{
						Name:    flagNameOpen,
						Aliases: []string{"o"},
						Value:   false,
						Usage:   "open and display vimrc.",
					},
				},
				Action: func(cCtx *cli.Context) error {
					// 何かしらオプションでない引数を渡されたらヘルプを出力して終了
					if cCtx.NumFlags() == 0 || cCtx.Args().Present() {
						cli.ShowSubcommandHelpAndExit(cCtx, 0)
					}

					// generate フラグがセットされていたら vimrc の再生成を行う
					if cCtx.Bool(flagNameGenerate) {
						err := os.WriteFile(vimrc, []byte(additionalVimrc), 0666)
						if err != nil {
							panic(err)
						}
						fmt.Printf("Generated additional vimrc to: %s\n", vimrc)
					}

					if cCtx.Bool(flagNameOpen) {
						util.OpenFileWithDefaultApp(vimrc)
						fmt.Printf("%s\n", vimrc)
					}

					return nil
				},
			},
			{
				Name:            "runargs",
				Usage:           "run subcommand's default arguments.",
				UsageText:       "devcontainer.vim runargs [OPTIONS...]",
				HideHelp:        false,
				SkipFlagParsing: false,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    flagNameGenerate,
						Aliases: []string{"g"},
						Value:   false,
						Usage:   "regenerate runargs file.",
					},
					&cli.BoolFlag{
						Name:    flagNameOpen,
						Aliases: []string{"o"},
						Value:   false,
						Usage:   "open and display runargs.",
					},
				},
				Action: func(cCtx *cli.Context) error {
					// 何かしらオプションでない引数を渡されたらヘルプを出力して終了
					if cCtx.NumFlags() == 0 || cCtx.Args().Present() {
						cli.ShowSubcommandHelpAndExit(cCtx, 0)
					}

					// generate フラグがセットされていたら runargs の再生成を行う
					if cCtx.Bool(flagNameGenerate) {
						err := os.WriteFile(runargs, []byte(runargsContent), 0666)
						if err != nil {
							panic(err)
						}
						fmt.Printf("Generated additional runargs to: %s\n", runargs)
					}

					if cCtx.Bool(flagNameOpen) {
						util.OpenFileWithDefaultApp(runargs)
						fmt.Printf("%s\n", runargs)
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
						Name:            tools.CdrFileName,
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
			{
				Name:      "clean",
				Usage:     "clean workspace cache files.",
				UsageText: "devcontainer.vim clean",
				Action: func(cCtx *cli.Context) error {

					// 実行確認
					var input string
					fmt.Printf("全ワークスペースのキャッシュを削除しますか？ [y/n] > ")
					fmt.Scan(&input)
					input = strings.TrimSpace(input)
					input = strings.ToLower(input)
					if input == "n" || input == "no" {
						return nil
					}

					// 削除処理
					err := os.RemoveAll(configDirForDocker)
					if err != nil {
						panic(err)
					}
					err = os.RemoveAll(configDirForDevcontainer)
					if err != nil {
						panic(err)
					}

					return nil
				},
			},
			{
				Name:            "index",
				Usage:           "Management index file",
				UsageText:       "devcontainer.vim index SUB_COMMAND",
				HideHelp:        false,
				SkipFlagParsing: false,
				Subcommands: []*cli.Command{
					{
						Name:      "update",
						Usage:     "Download newly index file",
						UsageText: "devcontainer.vim index update",
						Action: func(cCtx *cli.Context) error {

							// Features の一覧をダウンロード
							oras.Pull("ghcr.io/devcontainers/index", "latest", appCacheDir)

							return nil
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
func createConfigFile(devcontainerPath string, workspaceFolder string, configDirForDevcontainer string) (string, error) {
	// devcontainer の設定ファイルパス取得
	configFilePath, err := devcontainer.GetConfigurationFilePath(devcontainerPath, workspaceFolder)
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
