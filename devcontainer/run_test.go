package devcontainer

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestGetContainerArch(t *testing.T) {
	installCleanupTestCommands(t, t.TempDir())
	for _, tc := range []struct {
		name  string
		uname string
		want  string
	}{
		{"x86_64", " x86_64\r", "amd64"},
		{"amd64", "amd64", "amd64"},
		{"arm64", "arm64", "aarch64"},
		{"aarch64", "aarch64", "aarch64"},
		{"unsupported", "riscv64", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(runCleanupTestArchEnv, tc.uname)
			got, err := getContainerArch("test-container")
			if tc.want == "" {
				if got != "" || err == nil || !strings.Contains(err.Error(), "Unknown Architecture") {
					t.Fatalf("getContainerArch() = %q, %v, want an unsupported architecture error", got, err)
				}
			} else if got != tc.want || err != nil {
				t.Fatalf("getContainerArch() = %q, %v, want %q, nil", got, err, tc.want)
			}
		})
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
	install := exec.CommandContext(t.Context(), docker.Command, "exec", "--user", "root", containerID, "sh", "-c",
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
	const cidFileEnv = "DEVCONTAINER_VIM_RUN_TEST_CIDFILE"
	const started = "devcontainer.vim integration: VimEnter"
	if cidFile := os.Getenv(cidFileEnv); cidFile != "" {
		_, binDir, configDirForDocker, _ := createTempAppDirs(t)
		vimrc := filepath.Join(t.TempDir(), "vimrc")
		contents := "set nomore\nautocmd VimEnter * call writefile(['" + started + "'], '/dev/stdout', 'a') | qa!\n"
		if err := os.WriteFile(vimrc, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
		args := []string{"--cidfile", cidFile, bundledVimTestImage}
		if err := Run(args, true, true, true, "", binDir, false, "", configDirForDocker, vimrc, nil); err != nil {
			t.Fatalf("Run failed: %v", err)
		}
		return
	}

	requireLinuxDocker(t)
	script, err := exec.LookPath("script")
	if err != nil {
		t.Skip("script is required to provide a terminal for Docker exec")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cidFile := filepath.Join(t.TempDir(), "container-id")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	// Docker exec -it needs a terminal even when go test has redirected stdin.
	const testArg = "-test.run=^TestRunIntegration$"
	scriptArgs := []string{"-qefc", "exec " + shellQuote(self) + " " + testArg, "/dev/null"}
	if runtime.GOOS == "darwin" {
		scriptArgs = []string{"-q", "/dev/null", self, testArg}
	}
	cmd := exec.CommandContext(ctx, script, scriptArgs...)
	cmd.Env = append(os.Environ(), cidFileEnv+"="+cidFile, "SHELL=/bin/sh", "TERM=xterm")
	cmd.WaitDelay = 5 * time.Second
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		// The subprocess may have been interrupted before Run could stop the container.
		if id, readErr := os.ReadFile(cidFile); readErr == nil && len(id) != 0 {
			cleanupCtx, cancelCleanup := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancelCleanup()
			_ = exec.CommandContext(cleanupCtx, docker.Command, "rm", "-f", strings.TrimSpace(string(id))).Run()
		}
		t.Fatalf("Run did not finish: %v\n%s", ctx.Err(), output)
	}
	if err != nil {
		t.Fatalf("Run subprocess failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), started) {
		t.Fatalf("Vim did not report VimEnter:\n%s", output)
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
