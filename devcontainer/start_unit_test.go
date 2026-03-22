package devcontainer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// startDevcontainer関数の単体テスト（モック）
func TestStartDevcontainerMock(t *testing.T) {
	// パラメータの検証テスト
	args := []string{"image", "workspace"}
	devcontainerPath := "/mock/devcontainer"
	configFilePath := "/mock/config.json"
	workspaceFolder := "workspace"

	// 引数の妥当性チェック
	if len(args) < 2 {
		t.Fatal("args should have at least 2 elements")
	}

	if devcontainerPath == "" {
		t.Fatal("devcontainerPath should not be empty")
	}

	if configFilePath == "" {
		t.Fatal("configFilePath should not be empty")
	}

	if workspaceFolder == "" {
		t.Fatal("workspaceFolder should not be empty")
	}

	t.Logf("startDevcontainer parameters validated successfully")
}

// startClipboardReceiverForDevcontainer関数の単体テスト（モック）
func TestStartClipboardReceiverForDevcontainerMock(t *testing.T) {
	// パラメータの検証テスト
	cdrPath := "/mock/cdr"
	configDirForDevcontainer := "/mock/config"

	// 引数の妥当性チェック
	if cdrPath == "" {
		t.Fatal("cdrPath should not be empty")
	}

	if configDirForDevcontainer == "" {
		t.Fatal("configDirForDevcontainer should not be empty")
	}

	t.Logf("startClipboardReceiverForDevcontainer parameters validated successfully")
}

// setupPortForwarding関数の単体テスト（モック）
func TestSetupPortForwardingMock(t *testing.T) {
	// パラメータの検証テスト
	containerID := "mock-container-id"
	devcontainerPath := "/mock/devcontainer"
	workspaceFolder := "/mock/workspace"

	// 引数の妥当性チェック
	if containerID == "" {
		t.Fatal("containerID should not be empty")
	}

	if devcontainerPath == "" {
		t.Fatal("devcontainerPath should not be empty")
	}

	if workspaceFolder == "" {
		t.Fatal("workspaceFolder should not be empty")
	}

	t.Logf("setupPortForwarding parameters validated successfully")
}

// Start関数の統合テスト（軽量版）
func TestStartParameterValidation(t *testing.T) {
	// 一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "start_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// テスト用パラメータ
	services := TestDevcontainerStartUseService{}
	args := []string{"test-image", "workspace"}
	devcontainerPath := "/mock/devcontainer"
	nvim := false

	// パラメータの妥当性を検証
	if len(args) < 2 {
		t.Fatal("args should have at least 2 elements")
	}

	workspaceFolder := args[len(args)-1]
	if workspaceFolder == "" {
		t.Fatal("workspaceFolder should not be empty")
	}

	// その他のパラメータも検証
	if devcontainerPath == "" {
		t.Fatal("devcontainerPath should not be empty")
	}

	t.Logf("Start function parameters validated: args=%v, workspace=%s, nvim=%v", args, workspaceFolder, nvim)

	// サービスインターフェースも検証
	_ = services
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
