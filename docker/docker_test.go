package docker

import (
	"os/exec"
	"strings"
	"testing"
)

func TestPs(t *testing.T) {
	// DockerがインストールされていることをチェックするためのHelperコマンド
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// 基本的なdocker psコマンドがエラーなく実行できることをテスト
	result, err := Ps("status=running")
	if err != nil {
		// dockerデーモンが動いていない場合などはスキップ
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("Ps failed: %v", err)
	}

	// 結果が文字列であることを確認
	if result == "" {
		// 実行中のコンテナがない場合は空文字列が返る
		t.Logf("No running containers found")
	} else {
		// 結果にJSONの形式が含まれているかチェック（簡易）
		if !strings.Contains(result, "ID") {
			t.Logf("Unexpected output format: %s", result)
		}
	}
}

func TestPsWithInvalidFilter(t *testing.T) {
	// DockerがインストールされていることをチェックするためのHelperコマンド
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// 無効なフィルタでテスト
	result, err := Ps("invalid=filter")
	if err != nil {
		// dockerデーモンが動いていない場合などはスキップ
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skip("Docker daemon not running, skipping test")
		}
		// 無効なフィルタでもdockerコマンド自体は成功する場合がある
		t.Logf("Ps with invalid filter returned error: %v", err)
	}

	// 結果は空文字列になることが期待される
	if result != "" {
		t.Logf("Unexpected result with invalid filter: %s", result)
	}
}

func TestPsWithEmptyFilter(t *testing.T) {
	// DockerがインストールされていることをチェックするためのHelperコマンド
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// 空のフィルタでテスト
	result, err := Ps("")
	if err != nil {
		// dockerデーモンが動いていない場合などはスキップ
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("Ps with empty filter failed: %v", err)
	}

	// 結果が文字列であることを確認（空でも可）
	t.Logf("Ps with empty filter result: %s", result)
}

func TestPsWithValidLabel(t *testing.T) {
	// DockerがインストールされていることをチェックするためのHelperコマンド
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping test")
	}

	// ラベルフィルタでテスト（存在しないラベルを使用）
	result, err := Ps("label=devcontainer.local_folder=/nonexistent/path")
	if err != nil {
		// dockerデーモンが動いていない場合などはスキップ
		if strings.Contains(err.Error(), "Cannot connect to the Docker daemon") {
			t.Skip("Docker daemon not running, skipping test")
		}
		t.Fatalf("Ps with label filter failed: %v", err)
	}

	// 存在しないラベルなので結果は空文字列になることが期待される
	if result != "" {
		t.Logf("Unexpected result with nonexistent label: %s", result)
	}
}
