package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/host"
)

func generateConfig(cCtx *cli.Context) error {
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
			err := os.MkdirAll(filepath.Dir(configFilePath), 0755)
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
}

func (paths commandPaths) configureVimrc(cCtx *cli.Context) error {
	// 何かしらオプションでない引数を渡されたらヘルプを出力して終了
	if cCtx.NumFlags() == 0 || cCtx.Args().Present() {
		cli.ShowSubcommandHelpAndExit(cCtx, 0)
	}

	// generate フラグがセットされていたら vimrc の再生成を行う
	if cCtx.Bool(flagNameGenerate) {
		err := os.WriteFile(paths.vimrcPath, []byte(additionalVimrc), 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing vimrc: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated additional vimrc to: %s\n", paths.vimrcPath)
	}

	if cCtx.Bool(flagNameOpen) {
		err := host.OpenFileWithDefaultApp(paths.vimrcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening vimrc: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", paths.vimrcPath)
	}

	return nil
}

func (paths commandPaths) configureRunargs(cCtx *cli.Context) error {
	// 何かしらオプションでない引数を渡されたらヘルプを出力して終了
	if cCtx.NumFlags() == 0 || cCtx.Args().Present() {
		cli.ShowSubcommandHelpAndExit(cCtx, 0)
	}

	// generate フラグがセットされていたら runargs の再生成を行う
	if cCtx.Bool(flagNameGenerate) {
		err := os.WriteFile(paths.runargsPath, []byte(runargsContent), 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing runargs: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Generated additional runargs to: %s\n", paths.runargsPath)
	}

	if cCtx.Bool(flagNameOpen) {
		err := host.OpenFileWithDefaultApp(paths.runargsPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening runargs: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", paths.runargsPath)
	}

	return nil
}

func (paths commandPaths) cleanCache(*cli.Context) error {
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
	err := os.RemoveAll(paths.dockerConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error removing docker config directory: %v\n", err)
		}
		os.Exit(1)
	}
	err = os.RemoveAll(paths.devcontainerConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "Permission error: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error removing devcontainer config directory: %v\n", err)
		}
		os.Exit(1)
	}

	return nil
}
