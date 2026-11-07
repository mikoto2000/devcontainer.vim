package devcontainer

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/util"
)

// 段階的テスト: 各分割された関数を個別にテスト
func TestStartStepByStep(t *testing.T) {
	// devcontainerコマンドの存在確認
	_, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker command not found, skipping integration test")
	}

	// 設定ファイルの作成と検証
	t.Run("Config File Creation", func(t *testing.T) {
		_, _, _, configDirForDevcontainer := createTempAppDirs(t)

		devcontainerPath := requireTestBinary(t, "devcontainer")

		// 設定ファイルが作成できるか確認
		configFilePath, err := CreateConfigFile(devcontainerPath, "../test/project/TestStart", configDirForDevcontainer)
		if err != nil {
			// devcontainerコマンドが失敗する場合はスキップ
			if strings.Contains(err.Error(), "出力パースに失敗") {
				t.Skip("devcontainer CLI not working properly, skipping test")
			}
			t.Fatalf("Error creating config file: %v", err)
		}

		// 設定ファイルが存在することを確認
		if !util.IsExists(configFilePath) {
			t.Fatalf("Config file not created: %s", configFilePath)
		}

		t.Logf("Config file created successfully: %s", configFilePath)
	})
}

// エラーケーステスト
func TestStartErrorCases(t *testing.T) {
	t.Run("Empty args", func(t *testing.T) {
		cliPath := filepath.Join(t.TempDir(), "unused-devcontainer")
		err := Start(nil, nil, cliPath, true, true, true, "", "", false, "", "", "")
		if err == nil || !strings.Contains(err.Error(), "missing workspace folder") {
			t.Fatalf("Start with no arguments = %v, want a missing workspace error", err)
		}
	})
}
