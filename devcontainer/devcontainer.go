package devcontainer

import (
	_ "embed"

	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/dockercompose"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

//go:embed VimRun_system.template.sh
var vimRunX8664System string

//go:embed VimRun_x86_64_AppImage.template.sh
var vimRunX8664AppImage string

//go:embed VimRun_x86_64_static.template.sh
var vimRunX8664Static string

//go:embed VimRun_aarch64.template.sh
var vimRunAarch64 string

const containerCommand = "docker"

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
	workspaceFolder := args[len(args)-1]
	_, err := stopWorkspaceContainers(workspaceFolder, devcontainerPath, false)
	return err
}

func Down(args []string, devcontainerPath string, configDirForDevcontainer string) error {
	workspaceFolder := args[len(args)-1]
	removed, err := stopWorkspaceContainers(workspaceFolder, devcontainerPath, true)
	if err != nil || !removed {
		return err
	}

	configDir, err := util.GetConfigDir(configDirForDevcontainer, workspaceFolder)
	if err != nil {
		return err
	}

	if !hasNoCdrOption(args) {
		// clipboard-data-receiver を停止
		pidFile := filepath.Join(configDir, "pid")
		fmt.Printf("Read PID file: %s\n", pidFile)
		pidStringBytes, err := os.ReadFile(pidFile)
		if err != nil {
			return err
		}
		pid, err := strconv.Atoi(string(pidStringBytes))
		if err != nil {
			return err
		}
		fmt.Printf("clipboard-data-receiver PID: %d\n", pid)
		if err := tools.KillCdr(pid); err != nil {
			return fmt.Errorf("stop clipboard-data-receiver using PID file %q: %w", pidFile, err)
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

func GetConfigurationFilePath(devcontainerFilePath string, workspaceFolder string) (string, error) {
	stdout, _ := ReadConfiguration(devcontainerFilePath, "--workspace-folder", workspaceFolder)
	return GetConfigFilePath(stdout)
}

func ReadConfiguration(devcontainerFilePath string, readConfiguration ...string) (string, error) {
	args := append([]string{"read-configuration"}, readConfiguration...)
	result, err := Execute(devcontainerFilePath, args...)
	if err != nil {
		return "", errors.New("`devcontainer read-configuration` に失敗しました。`.devcontainer.json が存在することと、 docker エンジンが起動していることを確認してください。")
	}
	return result, err
}

func Templates(
	devcontainerFilePath string,
	workspaceFolder string,
	templateID string) (string, error) {
	// コマンドライン引数の末尾は `--workspace-folder` の値として使う

	args := []string{"templates", "apply", "--template-id", templateID, "--workspace-folder", workspaceFolder}
	return ExecuteCombineOutput(devcontainerFilePath, args...)
}

func Execute(devcontainerFilePath string, args ...string) (string, error) {
	fmt.Printf("run devcontainer: `%s %s`\n", devcontainerFilePath, strings.Join(args, " "))
	cmd := exec.Command(devcontainerFilePath, args...)
	stdout, err := cmd.Output()
	return string(stdout), err
}

func ExecuteCombineOutput(devcontainerFilePath string, args ...string) (string, error) {
	fmt.Printf("run devcontainer: `%s %s`\n", devcontainerFilePath, strings.Join(args, " "))
	cmd := exec.Command(devcontainerFilePath, args...)
	stdout, err := cmd.CombinedOutput()
	return string(stdout), err
}

// devcontainer.vim 起動時に使用する設定ファイルを作成する
// 設定ファイルは、 devcontainer.vim のキャッシュ内の `config` ディレクトリに、
// ワークスペースフォルダのパスを md5 ハッシュ化した名前のディレクトリに格納する.
func CreateConfigFile(devcontainerPath string, workspaceFolder string, configDirForDevcontainer string) (string, error) {
	// devcontainer の設定ファイルパス取得
	configFilePath, err := GetConfigurationFilePath(devcontainerPath, workspaceFolder)
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
