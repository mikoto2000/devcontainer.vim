package devcontainer

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/configstore"
)

//go:embed VimRun_system.template.sh
var vimRunX8664System string

//go:embed VimRun_x86_64_AppImage.template.sh
var vimRunX8664AppImage string

//go:embed VimRun_x86_64_static.template.sh
var vimRunX8664Static string

//go:embed VimRun_aarch64.template.sh
var vimRunAarch64 string

func GetConfigurationFilePath(devcontainerFilePath string, workspaceFolder string) (string, error) {
	stdout, err := ReadConfiguration(devcontainerFilePath, "--workspace-folder", workspaceFolder)
	if err != nil {
		return "", err
	}
	return GetConfigFilePath(stdout)
}

func ReadConfiguration(devcontainerFilePath string, readConfiguration ...string) (string, error) {
	args := append([]string{"read-configuration"}, readConfiguration...)
	result, err := Execute(devcontainerFilePath, args...)
	if err != nil {
		diagnostic := result
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			diagnostic += "\n" + string(exit.Stderr)
		}
		if detail := strings.TrimSpace(diagnostic); detail != "" {
			return "", fmt.Errorf("devcontainer read-configuration: %w: %s", err, detail)
		}
		return "", fmt.Errorf("devcontainer read-configuration: %w", err)
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
	mergedConfigFilePath, err := configstore.CreateConfigFileForDevcontainer(configDirForDevcontainer, workspaceFolder, configFilePath, additionalConfigurationFilePath)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("permission error: %w", err)
		}
		return "", err
	}

	fmt.Printf("Use configuration file: `%s`", mergedConfigFilePath)

	return mergedConfigFilePath, err
}
