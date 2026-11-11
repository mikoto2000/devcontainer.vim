//go:build !windows

package host

import (
	"os/exec"
	"runtime"
)

// OpenFileWithDefaultApp opens the file with its associated application.
func OpenFileWithDefaultApp(filePath string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	return exec.Command(opener, filePath).Run()
}
