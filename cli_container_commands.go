package main

import (
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func (paths commandPaths) runContainer(cCtx *cli.Context) error {
	args := cCtx.Args().Slice()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: devcontainer.vim run <IMAGE_OR_CONTAINER>")
		os.Exit(1)
	}

	// Requirements のチェック
	// 1. docker
	isExistsDocker := util.IsExistsCommand(containerCommand)
	if !isExistsDocker {
		fmt.Fprintf(os.Stderr, "docker not found.")
		os.Exit(1)
	}

	// シェル使用判定
	shell := ""
	if cCtx.String(flagNameShell) != "" {
		shell = cCtx.String(flagNameShell)
	} else if os.Getenv(envDevcontainerShellType) != "" {
		shell = os.Getenv(envDevcontainerShellType)
	}

	// cdr 使用判定
	noCdr := false
	if cCtx.String(flagNameNoCdr) != "" {
		noCdr = cCtx.Bool(flagNameNoCdr)
	}

	// port-forwarder 使用判定
	noPf := false
	if cCtx.String(flagNameNoPf) != "" {
		noPf = cCtx.Bool(flagNameNoPf)
	}

	// tmux 使用判定
	noTmux := false
	if cCtx.String(flagNameNoTmux) != "" {
		noTmux = cCtx.Bool(flagNameNoTmux)
	}

	var defaultRunargs []string
	// Windows では runargs を使用しない。
	if runtime.GOOS != "windows" {
		data, err := os.ReadFile(paths.runargsPath)
		if err != nil {
			return fmt.Errorf("read runargs: %w", err)
		}
		defaultRunargs, err = parseRunargs(string(data))
		if err != nil {
			return fmt.Errorf("parse runargs: %w", err)
		}
	}

	// 必要なファイルのダウンロード
	nvim := false
	if cCtx.Bool(flagNameNeoVim) || os.Getenv(envDevcontainerVimType) == "nvim" {
		nvim = true
	}
	cdrPath, err := tools.InstallRunTools(paths.binDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing run tools: %v\n", err)
		os.Exit(1)
	}

	err = devcontainer.Run(args, noCdr, noPf, noTmux, cdrPath, paths.binDir, nvim, shell, paths.dockerConfigDir, paths.vimrcPath, defaultRunargs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running docker: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) startContainer(cCtx *cli.Context) error {
	// devcontainer でコンテナを立てる

	// 引数チェック
	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
	args := cCtx.Args().Slice()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: missing workspace folder.\n")
		fmt.Fprintf(os.Stderr, "Usage: devcontainer.vim start <WORKSPACE_FOLDER>\n")
		os.Exit(1)
	}
	// 引数の末尾が存在するディレクトリかどうか確認
	if err := validateWorkspaceFolder(args[len(args)-1]); err != nil {
		return cli.Exit(fmt.Errorf("Error: %w", err), 1)
	}

	// シェル使用判定
	shell := ""
	if cCtx.String(flagNameShell) != "" {
		shell = cCtx.String(flagNameShell)
	} else if os.Getenv(envDevcontainerShellType) != "" {
		shell = os.Getenv(envDevcontainerShellType)
	}

	// cdr 使用判定
	noCdr := false
	if cCtx.String(flagNameNoCdr) != "" {
		noCdr = cCtx.Bool(flagNameNoCdr)
	}

	// port-forwarder 使用判定
	noPf := cCtx.Bool(flagNameNoPf)

	// tmux 使用判定
	noTmux := false
	if cCtx.String(flagNameNoTmux) != "" {
		noTmux = cCtx.Bool(flagNameNoTmux)
	}

	// 必要なファイルのダウンロード
	nvim := false
	if cCtx.Bool(flagNameNeoVim) || os.Getenv(envDevcontainerVimType) == "nvim" {
		nvim = true
	}
	devcontainerPath, cdrPath, err := tools.InstallStartTools(tools.DefaultInstallerUseServices{}, paths.binDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing start tools: %v\n", err)
		os.Exit(1)
	}

	workspaceFolder := args[len(args)-1]
	configFilePath, err := devcontainer.CreateConfigFile(devcontainerPath, workspaceFolder, paths.devcontainerConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error creating config file: %v\n", err)
		}
		os.Exit(1)
	}

	// devcontainer を用いたコンテナ立ち上げ
	err = devcontainer.Start(devcontainer.DefaultDevcontainerStartUseService{}, args, devcontainerPath, noCdr, noPf, noTmux, cdrPath, paths.binDir, nvim, shell, configFilePath, paths.vimrcPath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error executing devcontainer: %v\n", err)
		}
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) stopContainer(cCtx *cli.Context) error {
	// devcontainer でコンテナを stop

	// 引数チェック
	args := cCtx.Args().Slice()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: missing workspace folder.\n")
		fmt.Fprintf(os.Stderr, "Usage: devcontainer.vim stop <WORKSPACE_FOLDER>\n")
		os.Exit(1)
	}

	// 必要なファイルのダウンロード
	devcontainerPath, err := tools.InstallDevcontainerTools(paths.binDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error installing stop tools: %v\n", err)
		}
		os.Exit(1)
	}

	// devcontainer を用いたコンテナ終了
	err = devcontainer.Stop(args, devcontainerPath, paths.devcontainerConfigDir)
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
}

func (paths commandPaths) downContainer(cCtx *cli.Context) error {
	// devcontainer でコンテナを立てる

	// 引数チェック
	args := cCtx.Args().Slice()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: missing workspace folder.\n")
		fmt.Fprintf(os.Stderr, "Usage: devcontainer.vim down <WORKSPACE_FOLDER>\n")
		os.Exit(1)
	}
	if err := validateWorkspaceFolder(args[len(args)-1]); err != nil {
		return cli.Exit(fmt.Errorf("Error: %w", err), 1)
	}

	// 必要なファイルのダウンロード
	devcontainerPath, err := tools.InstallDevcontainerTools(paths.binDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Configuration file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error installing down tools: %v\n", err)
		}
		os.Exit(1)
	}

	// devcontainer を用いたコンテナ終了
	err = devcontainer.Down(args, devcontainerPath, paths.devcontainerConfigDir)
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
	workspaceFolder := args[len(args)-1]
	configDir, err := util.GetConfigDir(paths.cacheDir, workspaceFolder)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error getting configuration file path: %v\n", err)
		}
		os.Exit(1)
	}

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
}
