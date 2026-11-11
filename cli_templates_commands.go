package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/devcontainer"
	"github.com/mikoto2000/devcontainer.vim/v3/internal/host"
	"github.com/mikoto2000/devcontainer.vim/v3/oras"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
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

func downloadTemplateIndex(ctx context.Context, cacheDir string) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()
	return oras.Pull(ctx, "ghcr.io/devcontainers/index", "latest", cacheDir)
}

func (paths commandPaths) applyTemplate(cCtx *cli.Context) error {
	// 引数チェック
	args := cCtx.Args().Slice()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: devcontainer.vim templates apply <WORKSPACE_FOLDER>\n")
		os.Exit(1)
	}
	if err := validateWorkspaceFolder(args[len(args)-1]); err != nil {
		return cli.Exit(fmt.Errorf("Error: %w", err), 1)
	}

	// Features の一覧をダウンロード
	indexFileName := "devcontainer-index.json"
	indexFile := filepath.Join(paths.cacheDir, indexFileName)
	if !host.IsExists(indexFile) {
		fmt.Println("Download template index ... ")
		err := downloadTemplateIndex(cCtx.Context, paths.cacheDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading template index: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("done.")
	}

	var indexRoot IndexRoot
	jsonFile := filepath.Join(paths.cacheDir, indexFileName)
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
	devcontainerFilePath, err := tools.InstallDevcontainerTools(paths.binDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing template tools: %v\n", err)
		os.Exit(1)
	}

	// コマンドライン引数の末尾は `--workspace-folder` の値として使う
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
}

func (paths commandPaths) updateTemplateIndex(cCtx *cli.Context) error {
	// Features の一覧をダウンロード
	err := downloadTemplateIndex(cCtx.Context, paths.cacheDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Index file not found: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "Error updating index: %v\n", err)
		}
		os.Exit(1)
	}

	return nil
}
