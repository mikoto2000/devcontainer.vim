package host

import (
	"errors"
	"testing"

	"golang.org/x/sys/windows"
)

func TestOpenWindowsFilePassesLiteralPath(t *testing.T) {
	const path = `C:\config & notes^\日本語 %PATH%.vim`
	wantErr := errors.New("fixture association failed")
	called := false
	err := openWindowsFile(path, func(hwnd windows.Handle, verb, file, args, dir *uint16, show int32) error {
		called = true
		if got := windows.UTF16PtrToString(file); got != path {
			t.Errorf("ShellExecute file = %q, want %q", got, path)
		}
		if hwnd != 0 || verb != nil || args != nil || dir != nil || show != windows.SW_SHOWNORMAL {
			t.Error("expected the default shell verb with a file path only")
		}
		return wantErr
	})
	if !called || !errors.Is(err, wantErr) {
		t.Fatalf("shell called = %v, open error = %v, want %v", called, err, wantErr)
	}
}

func TestOpenWindowsFileRejectsEmbeddedNUL(t *testing.T) {
	err := openWindowsFile("name\x00suffix", func(windows.Handle, *uint16, *uint16, *uint16, *uint16, int32) error {
		t.Fatal("ShellExecute must not receive a truncated path")
		return nil
	})
	if err == nil {
		t.Fatal("expected a path conversion error")
	}
}
