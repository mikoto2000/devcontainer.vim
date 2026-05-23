package tools

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRunCdrTimesOutWhenStartupNeverCompletes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script based test is not supported on windows")
	}

	tempDir := t.TempDir()
	cdrPath := filepath.Join(tempDir, "fake-cdr.sh")
	script := "#!/bin/sh\nsleep 5\n"
	if err := os.WriteFile(cdrPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write fake cdr script: %v", err)
	}

	originalTimeout := cdrStartupTimeout
	originalInterval := cdrPollInterval
	cdrStartupTimeout = 200 * time.Millisecond
	cdrPollInterval = 20 * time.Millisecond
	defer func() {
		cdrStartupTimeout = originalTimeout
		cdrPollInterval = originalInterval
	}()

	_, _, err := RunCdr(cdrPath, tempDir)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !errors.Is(err, errCdrStartupTimeout) {
		t.Fatalf("expected timeout error, got: %v", err)
	}
}

func TestGetProcessInfoRejectsZeroPidOrPort(t *testing.T) {
	output := `{"pid":0,"address":"0.0.0.0","port":0}`
	pid, _, port, err := GetProcessInfo(output)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if pid != 0 || port != 0 {
		t.Fatalf("expected zero pid and port, got pid=%d port=%d", pid, port)
	}
}
