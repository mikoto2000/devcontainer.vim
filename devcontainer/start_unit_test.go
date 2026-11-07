package devcontainer

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

func TestStartDevcontainerPassesWorkspaceAndOptions(t *testing.T) {
	const workspace = "workspace with spaces"
	const override = "override with spaces.json"
	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{"workspace only", []string{workspace}, []string{"up", "--override-config", override, "--workspace-folder", workspace}},
		{"CLI options", []string{"--log-level", "debug", workspace}, []string{"up", "--log-level", "debug", "--override-config", override, "--workspace-folder", workspace}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cli, dir := installForwardTestCommands(t, "[]")
			id, err := startDevcontainer(cli, tc.args, override, workspace)
			if err != nil || id != "fixture-up-container" {
				t.Fatalf("startDevcontainer = %q, %v, want fixture-up-container", id, err)
			}
			assertForwardTestArgs(t, dir, "up-args", tc.want)
		})
	}
}

func TestStartClipboardReceiverForDevcontainer(t *testing.T) {
	dir := t.TempDir()
	cdr := installCleanupTestCommands(t, dir)
	pid, port, err := startClipboardReceiverForDevcontainer(cdr, dir)
	if err != nil {
		t.Fatal(err)
	}
	if pid > 0 {
		t.Cleanup(func() {
			if err := tools.KillCdr(pid); err != nil {
				t.Error(err)
			}
		})
	}
	if pid <= 0 || port <= 0 {
		t.Fatalf("clipboard receiver reported pid=%d port=%d", pid, port)
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		t.Fatalf("connect to clipboard receiver at %s: %v", address, err)
	}
	conn.Close()
}

func TestCreateStartVimCommandUsesScriptOnWsl(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "script")
	err := os.WriteFile(scriptPath, []byte("#!/bin/sh\n"), 0755)
	if err != nil {
		t.Fatalf("failed to create mock script command: %v", err)
	}

	t.Setenv("WSL_DISTRO_NAME", "Ubuntu")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cmd := createStartVimCommand(context.Background(), "/mock/devcontainer", []string{"exec", "--workspace-folder", ".", "/VimRun.sh"})

	if filepath.Base(cmd.Path) != "script" {
		t.Fatalf("expected script wrapper, got path: %s", cmd.Path)
	}
	if len(cmd.Args) != 4 {
		t.Fatalf("unexpected script args: %#v", cmd.Args)
	}
	if cmd.Args[1] != "-qefc" {
		t.Fatalf("expected -qefc, got: %s", cmd.Args[1])
	}
	if !strings.Contains(cmd.Args[2], "'/mock/devcontainer' 'exec' '--workspace-folder' '.' '/VimRun.sh'") {
		t.Fatalf("unexpected wrapped command: %s", cmd.Args[2])
	}
	if cmd.Args[3] != "/dev/null" {
		t.Fatalf("expected /dev/null output file, got: %s", cmd.Args[3])
	}
}

func TestCreateStartVimCommandUsesDirectExecOutsideWsl(t *testing.T) {
	originalValue, hadValue := os.LookupEnv("WSL_DISTRO_NAME")
	err := os.Unsetenv("WSL_DISTRO_NAME")
	if err != nil {
		t.Fatalf("failed to unset WSL_DISTRO_NAME: %v", err)
	}
	t.Cleanup(func() {
		if hadValue {
			_ = os.Setenv("WSL_DISTRO_NAME", originalValue)
		}
	})

	cmd := createStartVimCommand(context.Background(), "/mock/devcontainer", []string{"exec", "--workspace-folder", ".", "/VimRun.sh"})

	if cmd.Path != "/mock/devcontainer" {
		t.Fatalf("expected direct exec path, got: %s", cmd.Path)
	}
	if len(cmd.Args) != 5 {
		t.Fatalf("unexpected direct exec args: %#v", cmd.Args)
	}
	if cmd.Args[0] != "/mock/devcontainer" {
		t.Fatalf("expected command path as first arg, got: %#v", cmd.Args)
	}
	if cmd.Args[1] != "exec" {
		t.Fatalf("expected direct exec args, got: %#v", cmd.Args)
	}
}
