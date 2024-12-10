package devcontainer

import (
	"fmt"
	"os"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

func TestSetupContainer(t *testing.T) {
	appName := "devcontainer.vim"

	_, binDir, configDirForDocker, _, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		panic(err)
	}

	nvim := false
	cdrPath, err := tools.InstallRunTools(binDir, nvim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing run tools: %v\n", err)
		os.Exit(1)
	}

	vimrc := "../test/resource/TestRun/vimrc"

	setupContainer(
		[]string{},
		cdrPath,
		binDir,
		nvim,
		configDirForDocker,
		vimrc,
		[]string{},
	)

}
