package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

func (paths commandPaths) downloadVim(cCtx *cli.Context) error {
	// Vim のダウンロード
	_, err := tools.VIM(tools.DefaultInstallerUseServices{}).Install(paths.binDir, cCtx.String(flagNameArch), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing vim: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) downloadNeovim(cCtx *cli.Context) error {
	// NeoVim のダウンロード
	_, err := tools.NVIM(tools.DefaultInstallerUseServices{}).Install(paths.binDir, cCtx.String(flagNameArch), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing nvim: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) downloadTmux(cCtx *cli.Context) error {
	// tmux のダウンロード
	_, err := tools.Tmux(tools.DefaultInstallerUseServices{}).Install(paths.binDir, cCtx.String(flagNameArch), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing tmux: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) downloadDevcontainerCLI(*cli.Context) error {
	// devcontainer のダウンロード
	_, err := tools.DEVCONTAINER(tools.DefaultInstallerUseServices{}).Install(paths.binDir, "", true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing devcontainer: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) downloadClipboardReceiver(*cli.Context) error {
	// clipboard-data-receiver のダウンロード
	_, err := tools.CDR(tools.DefaultInstallerUseServices{}).Install(paths.binDir, "", true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing clipboard-data-receiver: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func (paths commandPaths) downloadPortForwarder(cCtx *cli.Context) error {
	// clipboard-data-receiver のダウンロード
	_, err := tools.PortForwarderContainer(tools.DefaultInstallerUseServices{}).Install(paths.binDir, cCtx.String(flagNameArch), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing port-forwarder: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func selfUpdate(*cli.Context) error {
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
}
