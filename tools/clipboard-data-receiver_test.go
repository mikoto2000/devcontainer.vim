package tools

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const nativeCdrTestEnv = "DEVCONTAINER_VIM_NATIVE_CDR_TEST"

func nativeCdrFixture(t *testing.T, mode string, timeout time.Duration) (string, string, string) {
	t.Helper()
	t.Setenv(nativeCdrTestEnv, mode)
	originalTimeout := cdrStartupTimeout
	originalInterval := cdrPollInterval
	cdrStartupTimeout = timeout
	cdrPollInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		cdrStartupTimeout = originalTimeout
		cdrPollInterval = originalInterval
	})
	executable := copyTestExecutable(t)
	dir := filepath.Dir(executable)
	return executable, filepath.Join(dir, "pid"), filepath.Join(dir, "port")
}

func TestRunCdrForNativeStarts(t *testing.T) {
	executable, pidFile, portFile := nativeCdrFixture(t, "ready", 5*time.Second)
	pid, port, err := runCdrForNative(executable, pidFile, portFile)
	if err != nil {
		t.Fatal(err)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Error(err)
		}
		process.Release()
	})
	if pid != readNativeCdrFixtureNumber(t, pidFile) || port != readNativeCdrFixtureNumber(t, portFile) {
		t.Fatalf("unexpected CDR process info: pid=%d port=%d", pid, port)
	}
}

func TestRunCdrTimesOutWhenStartupNeverCompletes(t *testing.T) {
	executable, pidFile, portFile := nativeCdrFixture(t, "timeout", 2*time.Second)
	pid, port, err := runCdrForNative(executable, pidFile, portFile)
	if !IsCdrStartupTimeout(err) || pid != 0 || port != 0 {
		t.Fatalf("expected timeout without process info, got pid=%d port=%d err=%v", pid, port, err)
	}
	if !strings.Contains(err.Error(), "waiting for CDR startup") {
		t.Errorf("timeout error lost subprocess diagnostics: %v", err)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(readNativeCdrFixtureNumber(t, portFile)))
	if conn, err := net.DialTimeout("tcp", address, 200*time.Millisecond); err == nil {
		conn.Close()
		t.Fatal("CDR was still listening after the startup timeout")
	}
}

func readNativeCdrFixtureNumber(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(string(data))
	if err != nil || number <= 0 {
		t.Fatalf("invalid fixture value in %s: %q", path, data)
	}
	return number
}

func TestMain(m *testing.M) {
	if mode := os.Getenv(nativeCdrTestEnv); mode != "" && len(os.Args) == 6 && os.Args[1] == "--pid-file" {
		if err := runNativeCdrFixture(mode, os.Args[2], os.Args[4]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runNativeCdrFixture(mode, pidFile, portFile string) error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	pid := os.Getpid()
	port := listener.Addr().(*net.TCPAddr).Port
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return err
	}
	if err := os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0600); err != nil {
		return err
	}
	if mode == "ready" {
		fmt.Printf("{\"pid\":%d,", pid)
		time.Sleep(40 * time.Millisecond)
		fmt.Printf("\"port\":%d}\n", port)
	} else {
		fmt.Fprintln(os.Stderr, "waiting for CDR startup")
	}
	// Bound the fixture lifetime even if the caller does not terminate it.
	time.Sleep(15 * time.Second)
	return nil
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
