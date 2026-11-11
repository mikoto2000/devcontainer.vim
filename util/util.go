// Package util preserves the public helpers used by existing v3 callers.
// Internal application code uses the packages that own each responsibility.
package util

import (
	"io/fs"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/configstore"
	"github.com/mikoto2000/devcontainer.vim/v3/internal/host"
)

type GetDirFunc = configstore.GetDirFunc

func IsExistsCommand(command string) bool {
	return host.IsExistsCommand(command)
}

func CreateConfigDirectory(pathFunc GetDirFunc, dirName string) (string, error) {
	return configstore.CreateConfigDirectory(pathFunc, dirName)
}

func CreateCacheDirectory(pathFunc GetDirFunc, dirName string) (string, string, string, string, error) {
	return configstore.CreateCacheDirectory(pathFunc, dirName)
}

func IsExists(filePath string) bool {
	return host.IsExists(filePath)
}

func AddExecutePermission(filePath string) error {
	return host.AddExecutePermission(filePath)
}

func ParseJwcc(jwccPath string) ([]byte, error) {
	return configstore.ParseJwcc(jwccPath)
}

func CreateConfigFileForDevcontainer(cacheDir, workspaceFolder, configFilePath, additionalConfigFilePath string) (string, error) {
	return configstore.CreateConfigFileForDevcontainer(cacheDir, workspaceFolder, configFilePath, additionalConfigFilePath)
}

func GetConfigDir(cacheDir, workspaceFolder string) (string, error) {
	return configstore.GetConfigDir(cacheDir, workspaceFolder)
}

func IsWsl() bool {
	return host.IsWsl()
}

func CreateFileWithContents(file, content string, permission fs.FileMode) error {
	return host.CreateFileWithContents(file, content, permission)
}

func NormalizeContainerArch(containerArch string) (string, error) {
	return host.NormalizeContainerArch(containerArch)
}

func OpenFileWithDefaultApp(filePath string) error {
	return host.OpenFileWithDefaultApp(filePath)
}

func RemoveEmptyString(input []string) []string {
	var result []string

	for _, v := range input {
		if strings.TrimSpace(v) != "" {
			result = append(result, v)
		}
	}

	return result
}
