package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

const containerCommand = "docker"

var version = "dev"

const envDevcontainerVimType = "DEVCONTAINER_VIM_TYPE"
const envDevcontainerShellType = "DEVCONTAINER_SHELL_TYPE"

const flagNameLicense = "license"
const flagNameNeoVim = "nvim"
const flagNameNoCdr = "nocdr"
const flagNameNoPf = "nopf"
const flagNameNoTmux = "notmux"
const flagNameShell = "shell"
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

func validateWorkspaceFolder(workspaceFolder string) error {
	fileInfo, err := os.Stat(workspaceFolder)
	if err != nil {
		return fmt.Errorf("workspace folder not found: %s", workspaceFolder)
	}
	if !fileInfo.IsDir() {
		return fmt.Errorf("workspace folder is not a directory: %s", workspaceFolder)
	}
	return nil
}

type commandPaths struct {
	cacheDir              string
	binDir                string
	dockerConfigDir       string
	devcontainerConfigDir string
	vimrcPath             string
	runargsPath           string
}

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
		fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
		os.Exit(1)
	}
	appCacheDir, binDir, configDirForDocker, configDirForDevcontainer, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating cache directory: %v\n", err)
		os.Exit(1)
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

	app := newCLI(commandPaths{
		cacheDir:              appCacheDir,
		binDir:                binDir,
		dockerConfigDir:       configDirForDocker,
		devcontainerConfigDir: configDirForDevcontainer,
		vimrcPath:             vimrc,
		runargsPath:           runargs,
	})

	// アプリ実行
	if err := app.Run(os.Args); err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error running devcontainer.vim: %v\n", err)
		}
		os.Exit(1)
	}
}

func newCLI(paths commandPaths) *cli.App {
	return &cli.App{
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
				Usage:              "show licenses.",
			},
			&cli.BoolFlag{
				Name:               flagNameNeoVim,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "use NeoVim.",
			},
			&cli.BoolFlag{
				Name:               flagNameNoCdr,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "disable clipboard-data-receiver.",
			},
			&cli.BoolFlag{
				Name:               flagNameNoPf,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "disable port-forwarder.",
			},
			&cli.BoolFlag{
				Name:               flagNameNoTmux,
				Value:              false,
				DisableDefaultText: true,
				Usage:              "disable tmux.",
			},
			&cli.StringFlag{
				Name:  flagNameShell,
				Value: "",
				Usage: "start with shell.",
			},
		},
		Action: showLicense,
		Commands: []*cli.Command{
			{
				Name:            "run",
				Usage:           "Run container use `docker run`",
				UsageText:       "devcontainer.vim run [DOCKER_OPTIONS...] [DOCKER_ARGS...]",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action:          paths.runContainer,
			},
			{
				Name:      "templates",
				Usage:     "Run `devcontainer templates`",
				UsageText: "devcontainer.vim templates <SUBCOMMAND>",
				Subcommands: []*cli.Command{
					{
						Name:      "apply",
						Usage:     "Apply template.",
						UsageText: "devcontainer.vim templates apply WORKSPACE_FOLDER",
						Action:    paths.applyTemplate,
					},
				},
			},
			{
				Name:            "start",
				Usage:           "Run `devcontainer up` and `devcontainer exec`",
				UsageText:       "devcontainer.vim start [DEVCONTAINER_OPTIONS...] WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action:          paths.startContainer,
			},
			{
				Name:            "stop",
				Usage:           "Stop devcontainers.",
				UsageText:       "devcontainer.vim stop WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action:          paths.stopContainer,
			},
			{
				Name:            "down",
				Usage:           "Stop and remove devcontainers.",
				UsageText:       "devcontainer.vim down WORKSPACE_FOLDER",
				HideHelp:        true,
				SkipFlagParsing: true,
				Action:          paths.downContainer,
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
				Action: generateConfig,
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
				Action: paths.configureVimrc,
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
				Action: paths.configureRunargs,
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
										Usage: "target CPU architecture.",
									},
								},
								Action: paths.downloadVim,
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
										Usage: "target CPU architecture.",
									},
								},
								Action: paths.downloadNeovim,
							},
						},
					},
					{
						Name:            "tmux",
						Usage:           "Management tmux",
						UsageText:       "devcontainer.vim tool tmux SUB_COMMAND",
						HideHelp:        false,
						SkipFlagParsing: false,
						Subcommands: []*cli.Command{
							{
								Name:            "download",
								Usage:           "Download newly tmux",
								UsageText:       "devcontainer.vim tool tmux download",
								HideHelp:        false,
								SkipFlagParsing: false,
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:  flagNameArch,
										Value: runtime.GOARCH,
										Usage: "target CPU architecture.",
									},
								},
								Action: paths.downloadTmux,
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
								Action:          paths.downloadDevcontainerCLI,
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
								Action:          paths.downloadClipboardReceiver,
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
										Usage: "target CPU architecture.",
									},
								},
								Action: paths.downloadPortForwarder,
							},
						},
					},
				},
			},
			{
				Name:      "clean",
				Usage:     "clean workspace cache files.",
				UsageText: "devcontainer.vim clean",
				Action:    paths.cleanCache,
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
						Action:    paths.updateTemplateIndex,
					},
				},
			},
			{
				Name:      "self-update",
				Usage:     "Update devcontainer.vim itself",
				UsageText: "devcontainer.vim self-update",
				Action:    selfUpdate,
			},
			{
				Name:      "bash-complete-func",
				Usage:     "Show bash complete func",
				UsageText: "devcontainer.vim bash-complete-func",
				Action:    printBashCompletion,
			},
		},
	}
}

func showLicense(cCtx *cli.Context) error {
	// ライセンスフラグが立っていればライセンスを表示して終
	if cCtx.Bool(flagNameLicense) {
		fmt.Println(license)
		fmt.Println()
		fmt.Println(notice)
		os.Exit(0)
	}

	return nil
}

func printBashCompletion(*cli.Context) error {
	fmt.Print(bash_complete_func)
	return nil
}
