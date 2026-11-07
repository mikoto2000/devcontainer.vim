package devcontainer

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
	"github.com/mikoto2000/devcontainer.vim/v3/tools"
)

// Debian Bookworm provides glibc for the bundled amd64 Vim AppImage.
const bundledVimTestImage = "debian:bookworm-slim"

// setupContainer関数のテスト（既存のテストを継続使用）
func TestSetupContainer(t *testing.T) {
	requireLinuxDocker(t)
	_, binDir, configDirForDocker, _ := createTempAppDirs(t)

	nvim := false
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	vimrc := "../test/resource/TestRun/vimrc"
	noCdr := false
	noPf := false

	setup, err := setupContainer(
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
	cleanupTestContainerSetup(t, setup)

	if err != nil {
		t.Fatalf("error: %s", err)
	}

	if setup.vimFileName != "vim" || setup.useSystemVim {
		t.Fatalf("setup selected %q (system=%v), want bundled Vim", setup.vimFileName, setup.useSystemVim)
	}
	if _, err := docker.Exec(setup.containerID, "test", "-x", "/vim"); err != nil {
		t.Fatalf("bundled Vim is not executable at /vim: %v", err)
	}
	//     /vimrc
	vimrcOutput, err := docker.Exec(setup.containerID, "sh", "-c", "ls /vimrc")
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
	requireLinuxDocker(t)

	args := []string{"alpine:latest"}
	defaultRunargs := []string{}

	containerID, err := startContainer(args, defaultRunargs)
	if err != nil {
		t.Fatalf("startContainer failed: %v", err)
	}

	// コンテナIDが返されることを確認
	if containerID == "" {
		t.Fatal("Container ID should not be empty")
	}

	cleanupTestContainerSetup(t, containerSetup{containerID: containerID})

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
	requireLinuxDocker(t)

	containerID, err := startContainer([]string{"debian:bookworm-slim"}, nil)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	cleanupTestContainerSetup(t, containerSetup{containerID: containerID})
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

// Run関数の統合テスト（実際のコンテナ使用）
func TestRunIntegration(t *testing.T) {
	requireLinuxDocker(t)
	_, binDir, configDirForDocker, _ := createTempAppDirs(t)

	nvim := false
	cdrPath := requireTestBinary(t, "clipboard-data-receiver")

	args := []string{bundledVimTestImage}
	vimrc := "../test/resource/TestRun/vimrc"
	defaultRunargs := []string{}
	noCdr := false
	noPf := false

	// Exercise setup directly so it finishes before test cleanup starts.
	setup, err := setupContainer(args, noCdr, noPf, false, cdrPath, binDir, nvim, configDirForDocker, vimrc, defaultRunargs)
	cleanupTestContainerSetup(t, setup)

	if err != nil {
		t.Fatalf("Run integration test failed: %v", err)
	}
	if setup.vimFileName != "vim" || setup.useSystemVim {
		t.Fatalf("setup selected %q (system=%v), want bundled Vim", setup.vimFileName, setup.useSystemVim)
	}
}

func cleanupTestContainerSetup(t *testing.T, setup containerSetup) {
	t.Helper()
	t.Cleanup(func() {
		if setup.cdrPID > 0 {
			if err := tools.KillCdr(setup.cdrPID); err != nil {
				t.Errorf("stop clipboard-data-receiver: %v", err)
			}
		}
		if setup.containerID != "" {
			if err := docker.Stop(setup.containerID); err != nil {
				t.Errorf("stop test container: %v", err)
			}
		}
	})
}
