package tools

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTmuxArchiveRootMember(t *testing.T) {
	for _, tt := range []struct {
		name            string
		members         []string
		want, wantError string
	}{
		{name: "root after nested", members: []string{"other/tmux", "tmux"}, want: "contents of tmux"},
		{name: "dot prefix", members: []string{"./tmux"}, want: "contents of ./tmux"},
		{name: "nested only", members: []string{"other/tmux"}, wantError: "target file not found at archive root"},
		{name: "duplicate root", members: []string{"tmux", "./tmux"}, wantError: "multiple target files"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "tmux.tar.gz")
			writeTmuxTestArchive(t, archive, tt.members)
			dest := filepath.Join(dir, "tmux")
			err := extractSingleFileFromTarGz(archive, "tmux", dest)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("extract error = %v, want %q", err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(dest)
			if err != nil || string(body) != tt.want {
				t.Fatalf("extracted %q, %v, want %q", body, err, tt.want)
			}
		})
	}
}

func writeTmuxTestArchive(t *testing.T, path string, members []string) {
	t.Helper()
	var data bytes.Buffer
	gzipWriter := gzip.NewWriter(&data)
	tarWriter := tar.NewWriter(gzipWriter)
	for _, name := range members {
		body := "contents of " + name
		header := &tar.Header{Name: name, Mode: 0644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tarWriter, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}
