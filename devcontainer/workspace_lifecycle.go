package devcontainer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/dockercompose"
	"github.com/mikoto2000/devcontainer.vim/v3/internal/configstore"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

type UnknownTypeError struct {
	msg string
}

func (e *UnknownTypeError) Error() string {
	return e.msg
}

func hasNoCdrOption(args []string) bool {
	for _, arg := range args {
		if arg == "--nocdr" {
			return true
		}
	}
	return false
}

func Stop(args []string, devcontainerPath string, configDirForDevcontainer string) error {
	if len(args) == 0 || args[len(args)-1] == "" {
		return errors.New("missing workspace folder")
	}
	workspaceFolder := args[len(args)-1]
	_, err := stopWorkspaceContainers(workspaceFolder, devcontainerPath, false)
	return err
}

func Down(args []string, devcontainerPath string, configDirForDevcontainer string) error {
	if len(args) == 0 || args[len(args)-1] == "" {
		return errors.New("missing workspace folder")
	}
	workspaceFolder := args[len(args)-1]
	removed, err := stopWorkspaceContainers(workspaceFolder, devcontainerPath, true)
	if err != nil || !removed {
		return err
	}

	configDir, err := configstore.GetConfigDir(configDirForDevcontainer, workspaceFolder)
	if err != nil {
		return err
	}

	if !hasNoCdrOption(args) {
		// clipboard-data-receiver を停止
		pidFile := filepath.Join(configDir, "pid")
		fmt.Printf("Read PID file: %s\n", pidFile)
		pidStringBytes, err := os.ReadFile(pidFile)
		switch {
		case err == nil:
			pid, err := strconv.Atoi(string(pidStringBytes))
			if err != nil {
				return err
			}
			fmt.Printf("clipboard-data-receiver PID: %d\n", pid)
			if err := tools.KillCdr(pid); err != nil {
				return fmt.Errorf("stop clipboard-data-receiver using PID file %q: %w", pidFile, err)
			}
		case !errors.Is(err, os.ErrNotExist):
			return err
		}
	}

	return os.RemoveAll(configDir)
}

// stopWorkspaceContainers reports whether a container or project was acted on.
func stopWorkspaceContainers(workspaceFolder, devcontainerPath string, remove bool) (bool, error) {
	output, err := ReadConfiguration(devcontainerPath, "--workspace-folder", workspaceFolder)
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(output) == "" {
		fmt.Printf("This directory is not a workspace for devcontainer: %s\n", workspaceFolder)
		return false, nil
	}
	result, err := UnmarshalReadConfigurationCommandResult([]byte(output))
	if err != nil {
		return false, err
	}
	config := result.Configuration
	if config.DockerComposeFile == nil {
		containerID, err := docker.GetContainerIDFromWorkspaceFolder(workspaceFolder)
		if err != nil {
			return false, err
		}
		if remove {
			fmt.Printf("Removing container %s.\n", containerID)
			return true, docker.Rm(containerID)
		}
		fmt.Printf("Stopping container %s.\n", containerID)
		return true, docker.Stop(containerID)
	}

	files, err := resolveComposeFiles(config)
	if err != nil {
		return false, err
	}
	project, err := dockercompose.FindProject(workspaceFolder, config.ConfigFilePath.FsPath, files...)
	if err != nil {
		return false, err
	}
	if project.Name == "" {
		fmt.Println("devcontainer already downed.")
		return false, nil
	}
	if remove {
		fmt.Printf("Removing Compose project %s.\n", project.Name)
		return true, project.Down()
	}
	fmt.Printf("Stopping Compose project %s.\n", project.Name)
	return true, project.Stop()
}

// resolveComposeFiles keeps the order used by Dev Containers when merging files.
func resolveComposeFiles(config Configuration) ([]string, error) {
	var files []string
	switch value := config.DockerComposeFile.(type) {
	case string:
		files = []string{value}
	case []any:
		if len(value) == 0 {
			return nil, &UnknownTypeError{msg: "dockerComposeFile must contain at least one path"}
		}
		for _, item := range value {
			file, ok := item.(string)
			if !ok {
				return nil, &UnknownTypeError{msg: "dockerComposeFile entries must be paths"}
			}
			files = append(files, file)
		}
	default:
		return nil, &UnknownTypeError{msg: "dockerComposeFile must be a path or an array of paths"}
	}

	for i, file := range files {
		if strings.TrimSpace(file) == "" {
			return nil, &UnknownTypeError{msg: "dockerComposeFile paths must not be empty"}
		}
		if !filepath.IsAbs(file) {
			if !filepath.IsAbs(config.ConfigFilePath.FsPath) {
				return nil, errors.New("devcontainer configuration path is missing or not absolute")
			}
			file = filepath.Join(filepath.Dir(config.ConfigFilePath.FsPath), file)
		}
		files[i] = filepath.Clean(file)
	}
	return files, nil
}
