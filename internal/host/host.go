// Package host provides local executable, filesystem, and platform operations.
package host

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func IsExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
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
	return os.Chmod(filePath, fileInfo.Mode()|0100)
}

func IsWsl() bool {
	if _, exists := os.LookupEnv("WSL_DISTRO_NAME"); exists {
		return true
	}
	// A WSL session can still launch Windows processes if the distribution
	// name was removed but its interop socket is available.
	socket := os.Getenv("WSL_INTEROP")
	if runtime.GOOS != "linux" || !filepath.IsAbs(socket) {
		return false
	}
	info, err := os.Stat(socket)
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func CreateFileWithContents(file string, content string, permission fs.FileMode) error {
	err := os.WriteFile(file, []byte(content), permission)
	if err != nil {
		return err
	}
	return nil
}

// NormalizeContainerArch returns the container cache key amd64 or aarch64.
// Empty input identifies a host-only executable.
func NormalizeContainerArch(containerArch string) (string, error) {
	if containerArch == "amd64" || containerArch == "x86_64" {
		return "amd64", nil
	} else if containerArch == "arm64" || containerArch == "aarch64" {
		return "aarch64", nil
	} else if containerArch == "" {
		return "", nil
	} else {
		return "", fmt.Errorf("Unknown Architecture: %q", containerArch)
	}
}
