package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/v3/oras"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
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

const containerCommand = "docker"

var version string

const envDevcontainerVimType = "DEVCONTAINER_VIM_TYPE"

const flagNameLicense = "license"
const flagNameNeoVim = "nvim"
const flagNameArch = "arch"

const flagNameGenerate = "generate"
const flagNameHome = "home"
const flagNameOutput = "output"
const flagNameOpen = "open"

//go:embed LICENSE
var license string

//go:embed NOTICE
var notice string

//go:embed bash_complete_func.bash
var bash_complete_func string

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
	appConfigDir, err := util.CreateConfigDirectory(os.UserConfigDir, appName)
	if err != nil {
		panic(err)
	}
	appCacheDir, binDir, configDirForDocker, configDirForDevcontainer, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		panic(err)
	}

	// vimrc ファイルの出力先を組み立て
	// vimrc を出力(既に存在するなら何もしない)
	vimrc := filepath.Join(appConfigDir, "vimrc")
	if !util.IsExists(vimrc) {
		err := util.CreateFileWithContents(vimrc, additionalVimrc, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating vimrc file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated additional vimrc to: %s\n", vimrc)
	}

	// runargs ファイルの出力先を組み立て
	// runargs を出力(既に存在するなら何もしない)
	runargs := filepath.Join(appConfigDir, "runargs")
	if !util.IsExists(runargs) {
		err := util.CreateFileWithContents(runargs, runargsContent, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating runargs file: %v\n", err)
			os.Exit(1)
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
			&cli.BoolFlag{
				Name:               flagNameNeoVim,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "use NeoVim.",
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
					isExistsDocker := util.IsExistsCommand(containerCommand)
					if !isExistsDocker {
						fmt.Fprintf(os.Stderr, "docker not found.")
						os.Exit(1)
					}

					// 必要なファイルのダウンロード

					nvim := false
					if cCtx.Bool(flagNameNeoVim) || os.Getenv(envDevcontainerVimType) == "nvim" {
						nvim = true
					}
					cdrPath, err := tools.InstallRunTools(binDir, nvim)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error installing run tools: %v\n", err)
						os.Exit(1)
					}

					// デフォルト引数取得
					defaultRunargsBytes, err := os.ReadFile(runargs)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error reading runargs: %v\n", err)
						os.Exit(1)
					}
					defaultRunargsString := string(defaultRunargsBytes)

					if runtime.GOOS == "windows" {
						// コンテナ起動
						// windows はシェル変数展開が上手くいかないので runargs を使用しない
						err = devcontainer.Run(cCtx.Args().Slice(), cdrPath, binDir, nvim, configDirForDocker, vimrc, []string{})
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error running docker: %v\n", err)
							os.Exit(1)
						}
					} else {
						// デフォルト引数内のシェル変数を展開
						extractedDofaultRunargsString, err := util.ExtractShellVariables(defaultRunargsString)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error extracting shell variables: %v\n", err)
							os.Exit(1)
						}

						// 展開したものを配列へ分割
						defaultRunargs, err := shlex.Split(extractedDofaultRunargsString, true)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error splitting runargs: %v\n", err)
							os.Exit(1)
						}

						// コンテナ起動
						err = devcontainer.Run(cCtx.Args().Slice(), cdrPath, binDir, nvim, configDirForDocker, vimrc, defaultRunargs)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error running docker: %v\n", err)
							os.Exit(1)
						}
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
								err := oras.Pull("ghcr.io/devcontainers/index", "latest", appCacheDir)
								if err != nil {
									fmt.Fprintf(os.Stderr, "Error downloading template index: %v\n", err)
									os.Exit(1)
								}
								fmt.Println("done.")
							}

							var indexRoot IndexRoot
							jsonFile := filepath.Join(appCacheDir, indexFileName)
							jsonData, err := os.ReadFile(jsonFile)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error reading template index: %v\n", err)
								os.Exit(1)
							}
							err = json.Unmarshal(jsonData, &indexRoot)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error unmarshalling template index: %v\n", err)
								os.Exit(1)
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
								fmt.Fprintf(os.Stderr, "Error running prompt: %v\n", err)
								os.Exit(1)
							}

							selectedItem := availableTemplateItems[i]

							// devcontainer の template サブコマンド実行

							// 必要なファイルのダウンロード
							devcontainerFilePath, err := tools.InstallTemplatesTools(binDir)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error installing template tools: %v\n", err)
								os.Exit(1)
							}

							// コマンドライン引数の末尾は `--workspace-folder` の値として使う
							args := cCtx.Args().Slice()
							workspaceFolder := args[len(args)-1]

							templateID := selectedItem.ID + ":" + selectedItem.Version

							// devcontainer を用いたコンテナ立ち上げ
							output, err := devcontainer.Templates(
								devcontainerFilePath,
								workspaceFolder,
								templateID)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error applying template: %v\n", err)
								os.Exit(1)
							}

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
					nvim := false
					if cCtx.Bool(flagNameNeoVim) || os.Getenv(envDevcontainerVimType) == "nvim" {
						nvim = true
					}
					devcontainerPath, cdrPath, err := tools.InstallStartTools(tools.DefaultInstallerUseServices{}, binDir)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error installing start tools: %v\n", err)
						os.Exit(1)
					}

					// コマンドライン引数の末尾は `--workspace-folder` の値として使う
					args := cCtx.Args().Slice()
					workspaceFolder := args[len(args)-1]
					configFilePath, err := createConfigFile(devcontainerPath, workspaceFolder, configDirForDevcontainer)
					if err != nil {
						if errors.Is(err, os.ErrNotExist) {
							fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
						}
						os.Exit(1)
					}

					// devcontainer を用いたコンテナ立ち上げ
					err = devcontainer.Start(args, devcontainerPath, cdrPath, binDir, nvim, configFilePath, vimrc)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error executing devcontainer: %v\n", err)
						}
						os.Exit(1)
					}

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
						if errors.Is(err, os.ErrNotExist) {
							fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error installing stop tools: %v\n", err)
						}
						os.Exit(1)
					}

					// devcontainer を用いたコンテナ終了
					err = devcontainer.Stop(cCtx.Args().Slice(), devcontainerPath, configDirForDevcontainer)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error stopping devcontainer: %v\n", err)
						}
						os.Exit(1)
					}

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
						if errors.Is(err, os.ErrNotExist) {
							fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error installing down tools: %v\n", err)
						}
						os.Exit(1)
					}

					// devcontainer を用いたコンテナ終了
					err = devcontainer.Down(cCtx.Args().Slice(), devcontainerPath, configDirForDevcontainer)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error downing devcontainer: %v\n", err)
						}
						os.Exit(1)
					}

					// 設定ファイルを削除
					// コマンドライン引数の末尾は `--workspace-folder` の値として使う
					args := cCtx.Args().Slice()
					workspaceFolder := args[len(args)-1]
					configDir := util.GetConfigDir(appCacheDir, workspaceFolder)

					fmt.Printf("Remove configuration file: `%s`\n", configDir)
					err = os.RemoveAll(configDir)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error removing configuration file: %v\n", err)
						}
						os.Exit(1)
					}

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
								fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
								os.Exit(1)
							}

							// 設定ファイルサンプルを出力
							err = os.WriteFile(configFilePath, []byte(devcontainerVimJSON), 0666)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
								os.Exit(1)
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
							fmt.Fprintf(os.Stderr, "Error writing vimrc: %v\n", err)
							os.Exit(1)
						}
						fmt.Printf("Generated additional vimrc to: %s\n", vimrc)
					}

					if cCtx.Bool(flagNameOpen) {
						err := util.OpenFileWithDefaultApp(vimrc)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error opening vimrc: %v\n", err)
							os.Exit(1)
						}
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
							fmt.Fprintf(os.Stderr, "Error writing runargs: %v\n", err)
							os.Exit(1)
						}
						fmt.Printf("Generated additional runargs to: %s\n", runargs)
					}

					if cCtx.Bool(flagNameOpen) {
						err := util.OpenFileWithDefaultApp(runargs)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error opening runargs: %v\n", err)
							os.Exit(1)
						}
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
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:  flagNameArch,
										Value: runtime.GOARCH,
										Usage: "download cpu archtecture.",
									},
								},
								Action: func(cCtx *cli.Context) error {

									// Vim のダウンロード
									_, err := tools.VIM(tools.DefaultInstallerUseServices{}).Install(binDir, cCtx.String(flagNameArch), true)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error installing vim: %v\n", err)
										os.Exit(1)
									}

									return nil
								},
							},
						},
					},
					{
						Name:            "nvim",
						Usage:           "Management nvim",
						UsageText:       "devcontainer.vim tool nvim SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly nvim",
								UsageText:       "devcontainer.vim tool nvim download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:  flagNameArch,
										Value: runtime.GOARCH,
										Usage: "download cpu archtecture.",
									},
								},
								Action: func(cCtx *cli.Context) error {

									// NeoVim のダウンロード
									_, err := tools.NVIM(tools.DefaultInstallerUseServices{}).Install(binDir, cCtx.String(flagNameArch), true)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error installing nvim: %v\n", err)
										os.Exit(1)
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
									_, err := tools.DEVCONTAINER(tools.DefaultInstallerUseServices{}).Install(binDir, "", true)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error installing devcontainer: %v\n", err)
										os.Exit(1)
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
								Usage:           "Download newly clipboard-data-receiver cli",
								UsageText:       "devcontainer.vim tool clipboard-data-receiver download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Action: func(cCtx *cli.Context) error {

									// clipboard-data-receiver のダウンロード
									_, err := tools.CDR(tools.DefaultInstallerUseServices{}).Install(binDir, "", true)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error installing clipboard-data-receiver: %v\n", err)
										os.Exit(1)
									}

									return nil
								},
							},
						},
					},
					{
						Name:            "port-forwarder",
						Usage:           "Management port-forwarder on container",
						UsageText:       "devcontainer.vim tool port-forwarder SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly port-forwarder cli",
								UsageText:       "devcontainer.vim tool port-forwarder download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:  flagNameArch,
										Value: runtime.GOARCH,
										Usage: "download cpu archtecture.",
									},
								},
								Action: func(cCtx *cli.Context) error {

									// clipboard-data-receiver のダウンロード
									_, err := tools.PortForwarderContainer(tools.DefaultInstallerUseServices{}).Install(binDir, cCtx.String(flagNameArch), true)
									if err != nil {
										fmt.Fprintf(os.Stderr, "Error installing port-forwarder: %v\n", err)
										os.Exit(1)
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
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error removing docker config directory: %v\n", err)
						}
						os.Exit(1)
					}
					err = os.RemoveAll(configDirForDevcontainer)
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error removing devcontainer config directory: %v\n", err)
						}
						os.Exit(1)
					}

					return nil
				},
			},
			{
				Name:            "index",
				Usage:           "Management dev container template index file",
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
							err := oras.Pull("ghcr.io/devcontainers/index", "latest", appCacheDir)
							if err != nil {
								if errors.Is(err, os.ErrNotExist) {
									fmt.Fprintf(os.Stderr, "Index file not found: %v\n", err)
								} else {
									fmt.Fprintf(os.Stderr, "Error updating index: %v\n", err)
								}
								os.Exit(1)
							}

							return nil
						},
					},
				},
			},
			{
				Name:      "self-update",
				Usage:     "Update devcontainer.vim itself",
				UsageText: "devcontainer.vim self-update",
				Action: func(cCtx *cli.Context) error {
					err := tools.SelfUpdate(tools.DefaultInstallerUseServices{})
					if err != nil {
						if errors.Is(err, os.ErrPermission) {
							fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
						} else {
							fmt.Fprintf(os.Stderr, "Error updating devcontainer.vim: %v\n", err)
						}
						os.Exit(1)
					}
					return nil
				},
			},
			{
				Name:      "bash-complete-func",
				Usage:     "Show bash complete func",
				UsageText: "devcontainer.vim bash-complete-func",
				Action: func(cCtx *cli.Context) error {
					fmt.Printf(bash_complete_func)
					return nil
				},
			},
		},
	})

	// アプリ実行
	err = devcontainerVimArgProcess.Run(os.Args)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error running devcontainer.vim: %v\n", err)
		}
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
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("configuration file not found: %w", err)
		}
		return "", err
	}

	// devcontainer.vim 用の追加設定ファイルを探す
	configurationFileName := configFilePath[:len(configFilePath)-len(filepath.Ext(configFilePath))]
	additionalConfigurationFilePath := configurationFileName + ".vim.json"

	// 設定管理フォルダに JSON を配置
	mergedConfigFilePath, err := util.CreateConfigFileForDevcontainer(configDirForDevcontainer, workspaceFolder, configFilePath, additionalConfigurationFilePath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("permission error: %w", err)
		}
		return "", err
	}

	fmt.Printf("Use configuration file: `%s`", mergedConfigFilePath)

	return mergedConfigFilePath, err
}
