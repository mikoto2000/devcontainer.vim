package util

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// OpenFileWithDefaultApp opens the file with its associated application.
func OpenFileWithDefaultApp(filePath string) error {
	return openWindowsFile(filePath, windows.ShellExecute)
}

func openWindowsFile(filePath string, shellExecute func(windows.Handle, *uint16, *uint16, *uint16, *uint16, int32) error) error {
	file, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return err
	}
	if err := shellExecute(0, nil, file, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("open %q: %w", filePath, err)
	}
	return nil
}
