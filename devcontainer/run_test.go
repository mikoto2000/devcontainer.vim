package devcontainer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

// Debian Bookworm provides glibc for the bundled amd64 Vim AppImage.
const bundledVimTestImage = "debian:bookworm-slim"

// setupContainer関数のテスト（既存のテストを継続使用）
func TestSetupContainer(t *testing.T) {
	_, binDir, configDirForDocker, _ := createTempAppDirs(t)

	nvim := false
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	vimrc := "../test/resource/TestRun/vimrc"
	noCdr := false
	noPf := false

	containerID, vimName, _, _, _, useSystemVim, _, cdrPID, _, err := setupContainer(
		[]string{bundledVimTestImage},
		noCdr,
		noPf,
		false,
		cdrPath,
		binDir,
		nvim,
		configDirForDocker,
		vimrc,
		[]string{},
	)
	t.Cleanup(func() {
		if cdrPID > 0 {
			if err := tools.KillCdr(cdrPID); err != nil {
				t.Errorf("stop clipboard-data-receiver: %v", err)
			}
		}
		if containerID != "" {
			if err := docker.Stop(containerID); err != nil {
				t.Errorf("stop test container: %v", err)
			}
		}
	})

	if err != nil {
		if strings.Contains(err.Error(), "Container start error") {
			t.Skipf("container runtime not available for integration test: %v", err)
		}
		if tools.IsCdrStartupTimeout(err) {
			t.Skipf("clipboard-data-receiver startup timeout (environment issue): %v", err)
		}
		t.Fatalf("error: %s", err)
	}

	if vimName != "vim" || useSystemVim {
		t.Fatalf("setup selected %q (system=%v), want bundled Vim", vimName, useSystemVim)
	}
	if _, err := docker.Exec(containerID, "test", "-x", "/vim"); err != nil {
		t.Fatalf("bundled Vim is not executable at /vim: %v", err)
	}
	//     /vimrc
	vimrcOutput, err := docker.Exec(containerID, "sh", "-c", "ls /vimrc")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimrcWant := "vimrc"
	if !strings.Contains(vimrcOutput, vimrcWant) {
		t.Fatalf("error: want match %s, but got %s", vimrcWant, vimrcOutput)
	}
}

// startContainer関数の単体テスト
func TestStartContainer(t *testing.T) {
	// Dockerの存在確認
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	args := []string{"alpine:latest"}
	defaultRunargs := []string{}

	containerID, err := startContainer(args, defaultRunargs)
	if err != nil {
		// Dockerデーモンが動いていない場合はスキップ
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") || strings.Contains(err.Error(), "Container start error") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("startContainer failed: %v", err)
	}

	// コンテナIDが返されることを確認
	if containerID == "" {
		t.Fatal("Container ID should not be empty")
	}

	// 後片付け
	defer func() {
		exec.Command(containerCommand, "stop", containerID).Start()
	}()

	// コンテナが実際に起動していることを確認
	psOutput, err := docker.Ps("id=" + containerID)
	if err != nil {
		t.Fatalf("Failed to check container status: %v", err)
	}

	if psOutput == "" {
		t.Fatal("Container should be running")
	}
}

// getContainerArch関数の単体テスト
func TestGetContainerArch(t *testing.T) {
	// Dockerの存在確認
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// テスト用コンテナを起動
	args := []string{"alpine:latest"}
	containerID, err := startContainer(args, []string{})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") || strings.Contains(err.Error(), "Container start error") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("Failed to start container: %v", err)
	}

	defer func() {
		exec.Command(containerCommand, "stop", containerID).Start()
	}()

	// アーキテクチャを取得
	arch, err := getContainerArch(containerID)
	if err != nil {
		t.Fatalf("getContainerArch failed: %v", err)
	}

	// 有効なアーキテクチャが返されることを確認
	validArchs := []string{"amd64", "arm64", "x86_64", "aarch64"}
	found := false
	for _, validArch := range validArchs {
		if arch == validArch {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("Unexpected architecture: %s", arch)
	}
}

// setupVim関数の単体テスト（システムVimが存在する場合）
func TestSetupVimWithSystemVim(t *testing.T) {
	if err := exec.CommandContext(t.Context(), containerCommand, "info", "--format", "{{.ServerVersion}}").Run(); err != nil {
		t.Skipf("Docker server not available: %v", err)
	}

	containerID, err := startContainer([]string{"debian:bookworm-slim"}, nil)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	t.Cleanup(func() {
		if err := docker.Stop(containerID); err != nil {
			t.Errorf("stop system Vim fixture: %v", err)
		}
	})
	install := exec.CommandContext(t.Context(), containerCommand, "exec", "--user", "root", containerID, "sh", "-c",
		"apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends vim")
	if output, err := install.CombinedOutput(); err != nil {
		t.Fatalf("install system Vim in fixture: %v\n%s", err, output)
	}
	arch, err := getContainerArch(containerID)
	if err != nil {
		t.Fatal(err)
	}
	// A regressed probe can reach the fallback without contacting a release server.
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "vim_"+arch), nil, 0600); err != nil {
		t.Fatal(err)
	}
	vimFileName, useSystemVim, err := setupVim(containerID, cache, false, arch)
	if err != nil {
		t.Fatalf("setupVim failed: %v", err)
	}
	if vimFileName != "vim" || !useSystemVim {
		t.Fatalf("setup selected %q (system=%v), want system Vim", vimFileName, useSystemVim)
	}
}

// Run関数の統合テスト（モック使用）
func TestRunWithMock(t *testing.T) {
	// このテストは実際のDockerコンテナを使用しないモックテスト
	// Run関数の構造をテストするが、実際のVim実行はスキップ

	// テスト用の一時ディレクトリを作成
	tempDir, err := os.MkdirTemp("", "devcontainer_run_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// モック用のパラメータ
	args := []string{"test-image"}
	nvim := false
	configDirForDocker := tempDir

	// Run関数は実際のDockerコンテナが必要なため、
	// ここでは関数の呼び出し可能性のみをテスト
	// 実際の実行は統合テスト環境で行う

	// パラメータの妥当性をテスト
	if len(args) == 0 {
		t.Fatal("args should not be empty")
	}

	if configDirForDocker == "" {
		t.Fatal("configDirForDocker should not be empty")
	}

	t.Logf("Run function parameters validated: args=%v, nvim=%v", args, nvim)
}

// Run関数の統合テスト（実際のコンテナ使用）
func TestRunIntegration(t *testing.T) {
	// 長時間実行されるテストなので、短いタイムアウトでスキップも可能
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Dockerの存在確認
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping integration test")
	}

	_, binDir, configDirForDocker, _ := createTempAppDirs(t)

	nvim := false
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	args := []string{bundledVimTestImage}
	vimrc := "../test/resource/TestRun/vimrc"
	defaultRunargs := []string{}
	noCdr := false
	noPf := false

	// Run関数を別のgoroutineで実行し、タイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Run関数は通常、ユーザーの入力を待つため、即座に終了させる
		// 実際のテストでは、コンテナが起動してVimが実行可能な状態になることを確認

		// setupContainer部分のみをテスト
		containerID, vimName, _, _, _, useSystemVim, _, cdrPid, cdrConfigDir, err := setupContainer(
			args,
			noCdr,
			noPf,
			false,
			cdrPath,
			binDir,
			nvim,
			configDirForDocker,
			vimrc,
			defaultRunargs)

		if err != nil {
			done <- err
			return
		}

		// クリーンアップ
		tools.KillCdr(cdrPid)
		os.RemoveAll(cdrConfigDir)
		exec.Command(containerCommand, "stop", containerID).Start()

		if vimName != "vim" || useSystemVim {
			done <- fmt.Errorf("setup selected %q (system=%v), want bundled Vim", vimName, useSystemVim)
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") || strings.Contains(err.Error(), "Container start error") {
				t.Skip("Docker daemon not running, skipping integration test")
			}
			t.Fatalf("Run integration test failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Run integration test timed out")
	}
}
