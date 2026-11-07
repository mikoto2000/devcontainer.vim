package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type TestInstallerUseServices struct{}

func (s TestInstallerUseServices) GetLatestReleaseFromGitHub(owner string, repository string) (string, error) {
	return "", nil
}

func (s TestInstallerUseServices) Download(downloadURL string, destPath string) error {
	return os.WriteFile(destPath, []byte{}, 0755)
}

func TestInstallStartTools(t *testing.T) {
	binDir := t.TempDir()
	services := TestInstallerUseServices{}
	devcontainerPath, cdrPath, err := InstallStartTools(services, binDir)
	if err != nil {
		t.Fatalf("Error installing start tools: %v", err)
	}

	// devcontainer の存在確認
	wantDevcontainerPath := filepath.Join(binDir, DEVCONTAINER(services).FileName)
	if wantDevcontainerPath != devcontainerPath {
		t.Fatalf("want %s, but got %s", wantDevcontainerPath, devcontainerPath)
	}
	_, err = os.Stat(wantDevcontainerPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
	// cdr の存在確認
	wantCdrPath := filepath.Join(binDir, CDR(services).FileName)
	if wantCdrPath != cdrPath {
		t.Fatalf("want %s, but got %s", wantCdrPath, cdrPath)
	}
	_, err = os.Stat(wantCdrPath)
	if err != nil {
		t.Fatalf("error: %s", err)
	}
}

func TestInstallStartToolsPreservesExistingTestDirectory(t *testing.T) {
	// Model a repository root without putting the real test fixtures at risk.
	root := t.TempDir()
	fixture := filepath.Join(root, "test", "project", "devcontainer.json")
	const contents = "{\"name\":\"existing fixture\"}\n"
	if err := os.MkdirAll(filepath.Dir(fixture), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixture, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}

	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "-test.run=^TestInstallStartTools$", "-test.v")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("TestInstallStartTools failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "--- PASS: TestInstallStartTools ") {
		t.Fatalf("TestInstallStartTools did not run:\n%s", output)
	}
	if data, err := os.ReadFile(fixture); err != nil || string(data) != contents {
		t.Fatalf("existing test fixture after TestInstallStartTools = %q, %v, want %q", data, err, contents)
	}
}

func TestDownloadPreservesExecutableOnFailure(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		length string
		body   string
		want   string
	}{
		{name: "not found", status: http.StatusNotFound, body: "missing", want: "404 Not Found"},
		{name: "server error", status: http.StatusInternalServerError, body: "error", want: "500 Internal Server Error"},
		{name: "partial response", status: http.StatusPartialContent, body: "partial", want: "206 Partial Content"},
		{name: "truncated body", status: http.StatusOK, length: "100", body: "partial", want: io.ErrUnexpectedEOF.Error()},
		{name: "empty body", status: http.StatusOK, want: "empty response body"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			executablePath := copyTestExecutable(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.length != "" {
					w.Header().Set("Content-Length", tt.length)
				}
				w.WriteHeader(tt.status)
				io.WriteString(w, tt.body)
			}))
			defer server.Close()

			err := download(server.URL, executablePath)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("download error = %v, want %q", err, tt.want)
			}
			assertExecutableRuns(t, executablePath)
		})
	}
}

func TestDownloadPreservesExecutableOnTimeout(t *testing.T) {
	for _, phase := range []string{"headers", "body"} {
		t.Run(phase, func(t *testing.T) {
			executablePath := copyTestExecutable(t)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if phase == "body" {
					w.Header().Set("Content-Length", "100")
					io.WriteString(w, "partial")
					w.(http.Flusher).Flush()
				}
				// Keep the fixture bounded even if the downloader stops canceling requests.
				limit := time.NewTimer(4 * time.Second)
				defer limit.Stop()
				select {
				case <-r.Context().Done():
				case <-limit.C:
				}
			}))
			defer server.Close()

			client := &http.Client{Timeout: 500 * time.Millisecond}
			err := downloadWithClient(client, server.URL, executablePath)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("download error = %v, want a deadline error", err)
			}
			var requestErr *url.Error
			if gotHeadersError := errors.As(err, &requestErr); gotHeadersError != (phase == "headers") {
				t.Errorf("download error during %s = %v", phase, err)
			}
			assertExecutableRuns(t, executablePath)
		})
	}
}

func TestToolInstallRejectsUnknownArchitecture(t *testing.T) {
	dir := t.TempDir()
	tool := Tool{
		FileName: "test-tool",
		CalculateDownloadURL: func(arch string) (string, error) {
			t.Fatalf("attempted to resolve a download URL for %q", arch)
			return "", nil
		},
	}
	path, err := tool.Install(dir, "riscv64", true)
	if path != "" || err == nil || !strings.Contains(err.Error(), "Unknown Architecture") {
		t.Fatalf("Install returned %q, %v; want an empty path and an architecture error", path, err)
	}
}

func TestToolInstallPreservesExecutableOnFailure(t *testing.T) {
	executablePath := copyTestExecutable(t)
	installErr := errors.New("install interrupted")
	tool := Tool{
		FileName:             filepath.Base(executablePath),
		CalculateDownloadURL: func(string) (string, error) { return "", nil },
		installFunc: func(_ func(string, string) error, _, path, _ string) (string, error) {
			if err := os.WriteFile(path, []byte("partial"), 0755); err != nil {
				return "", err
			}
			return path, installErr
		},
	}

	if _, err := tool.Install(filepath.Dir(executablePath), "", true); !errors.Is(err, installErr) {
		t.Fatalf("Install error = %v, want %v", err, installErr)
	}
	assertExecutableRuns(t, executablePath)
}

func TestUpdateRejectsInvalidExecutable(t *testing.T) {
	executablePath := copyTestExecutable(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html>not an executable</html>")
	}))
	defer server.Close()

	if err := updateExecutable(download, server.URL, executablePath); err == nil || !strings.Contains(err.Error(), "read update executable build info") {
		t.Fatalf("updateExecutable error = %v, want invalid executable error", err)
	}
	assertExecutableRuns(t, executablePath)
}

type selfUpdateTestServices struct {
	TestInstallerUseServices
	targetPath string
}

func (s selfUpdateTestServices) Download(downloadURL, destPath string) error {
	stageDir := filepath.Dir(destPath)
	if filepath.Dir(stageDir) != filepath.Dir(s.targetPath) || filepath.Base(destPath) != filepath.Base(s.targetPath) {
		return fmt.Errorf("update staged at %q, want a staging file for %q", destPath, s.targetPath)
	}
	return s.TestInstallerUseServices.Download(downloadURL, destPath)
}

func TestSelfUpdateUsesTemporaryExecutable(t *testing.T) {
	target := copyTestExecutable(t)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}

	// The stub writes an empty file, as in the original self-update test.
	services := selfUpdateTestServices{targetPath: target}
	if err := selfUpdate(services, target); err == nil || !strings.Contains(err.Error(), "read update executable build info") {
		t.Errorf("selfUpdate error = %v, want empty staged executable to be rejected", err)
	}

	for _, path := range []string{binary, target} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, original) {
			t.Errorf("self-update changed %q", path)
		}
		assertExecutableRuns(t, path)
	}
}

func TestReplaceExecutableWithBackupOnFailure(t *testing.T) {
	for _, name := range []string{"back up", "replace", "restore"} {
		t.Run(name, func(t *testing.T) {
			executablePath := copyTestExecutable(t)
			stagedPath := filepath.Join(filepath.Dir(executablePath), "staged.exe")
			backupErr := errors.New("backup failed")
			replaceErr := errors.New("replacement failed")
			restoreErr := errors.New("restoration failed")
			var backupPath string
			rename := func(from, to string) error {
				switch from {
				case executablePath:
					backupPath = to
					if name == "back up" {
						return backupErr
					}
				case stagedPath:
					return replaceErr
				case backupPath:
					if name == "restore" {
						return restoreErr
					}
				}
				return os.Rename(from, to)
			}

			err := replaceExecutableWithBackup(stagedPath, executablePath, rename)
			if name == "back up" {
				if !errors.Is(err, backupErr) {
					t.Fatalf("backup error = %v, want %v", err, backupErr)
				}
			} else if !errors.Is(err, replaceErr) {
				t.Fatalf("replacement error = %v, want %v", err, replaceErr)
			}
			if name == "restore" {
				if !errors.Is(err, restoreErr) || !strings.Contains(err.Error(), filepath.Base(backupPath)) {
					t.Fatalf("expected restoration error and backup path, got %v", err)
				}
				assertExecutableRuns(t, backupPath)
				return
			}
			assertExecutableRuns(t, executablePath)
		})
	}
}

func copyTestExecutable(t *testing.T) string {
	t.Helper()
	currentPath, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "installed.exe")
	if err := os.WriteFile(path, binary, 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertExecutableRuns(t *testing.T, path string) {
	t.Helper()
	if output, err := exec.Command(path, "-test.run=^$").CombinedOutput(); err != nil {
		t.Fatalf("could not start %s: %v\n%s", path, err, output)
	}
}
