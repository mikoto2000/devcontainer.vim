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

