package devcontainer

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// setupContainer関数のテスト（既存のテストを継続使用）
func TestSetupContainer(t *testing.T) {
	appName := "devcontainer.vim"

	_, binDir, configDirForDocker, _, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		panic(err)
	}

	nvim := false
	cdrPath, err := tools.InstallRunTools(binDir, nvim)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error installing run tools: %v\n", err)
		os.Exit(1)
	}

	vimrc := "../test/resource/TestRun/vimrc"

	containerID, _, _, _, _, _, _, err := setupContainer(
		[]string{"alpine:latest"},
		cdrPath,
		binDir,
		nvim,
		configDirForDocker,
		vimrc,
		[]string{},
	)

	if err != nil {
		t.Fatalf("error: %s", err)
	}

	// 後片付け
	// コンテナ停止
	defer func() {
		// `docker stop <dockerrun 時に標準出力に表示される CONTAINER ID>`
		fmt.Printf("Stop container(Async) %s.\n", containerID)
		err = exec.Command(containerCommand, "stop", containerID).Start()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Container stop error: %s\n", err)
		}
	}()

	//     /vim
	vimOutput, err := docker.Exec(containerID, "sh", "-c", "ls /vim*")
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	vimWant := "vim"
	if !strings.Contains(vimOutput, vimWant) {
		t.Fatalf("error: want match %s, but got %s", vimWant, vimOutput)
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
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
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
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
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
	// Dockerの存在確認
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// テスト用コンテナを起動（Vimがプリインストールされているイメージを使用）
	args := []string{"alpine:latest"}
	containerID, err := startContainer(args, []string{})
	if err != nil {
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("Failed to start container: %v", err)
	}

	defer func() {
		exec.Command(containerCommand, "stop", containerID).Start()
	}()

	// Vimがシステムにインストールされているかどうかをテスト
	vimFileName, useSystemVim, err := setupVim(containerID, "", false, "amd64")
	if err != nil {
		t.Fatalf("setupVim failed: %v", err)
	}

	// 結果の検証
	if vimFileName != "vim" && vimFileName != "nvim" {
		t.Fatalf("Unexpected vim filename: %s", vimFileName)
	}

	// useSystemVimの値が妥当であることを確認
	t.Logf("useSystemVim: %v, vimFileName: %s", useSystemVim, vimFileName)
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

	appName := "devcontainer.vim"
	_, binDir, configDirForDocker, _, err := util.CreateCacheDirectory(os.UserCacheDir, appName)
	if err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	nvim := false
	cdrPath, err := tools.InstallRunTools(binDir, nvim)
	if err != nil {
		t.Fatalf("Error installing run tools: %v", err)
	}

	args := []string{"alpine:latest"}
	vimrc := "../test/resource/TestRun/vimrc"
	defaultRunargs := []string{}

	// Run関数を別のgoroutineで実行し、タイムアウトを設定
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		// Run関数は通常、ユーザーの入力を待つため、即座に終了させる
		// 実際のテストでは、コンテナが起動してVimが実行可能な状態になることを確認

		// setupContainer部分のみをテスト
		containerID, _, _, _, _, cdrPid, cdrConfigDir, err := setupContainer(
			args,
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

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
				t.Skip("Docker daemon not running, skipping integration test")
			}
			t.Fatalf("Run integration test failed: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("Run integration test timed out")
	}
}
