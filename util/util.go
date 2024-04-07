package util

import (
	"os"
	"os/exec"
	"path/filepath"
)

func IsExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		return false
	}
	return true
}

type GetDirFunc func() (string, error)

func CreateDirectory(pathFunc GetDirFunc, dirName string) string {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appCacheDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, 0766); err != nil {
		panic(err)
	}
	return appCacheDir
}

func IsExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func AddExecutePermission(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	fileMode := fileInfo.Mode()
	err = os.Chmod(filePath, fileMode|0111)
	if err != nil {
		return err
	}

	return nil
}

// TODO: 実装
func CreateConfigFileForDevcontainerVim(cacheDir string, configFilePath string, additionalConfigFilePath string) (string, error) {

	// TODO: 設定管理フォルダに JSON を配置
	return configFilePath, nil
}
