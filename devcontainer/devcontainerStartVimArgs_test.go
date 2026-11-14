package devcontainer

import (
	"slices"
	"testing"
)

func TestBuildDevcontainerStartVimExecArgsForVimRunScript(t *testing.T) {
	args := buildDevcontainerStartVimExecArgs("test-container", "/workspace", "/cache with spaces/override.json", "")

	if !slices.Equal(args, []string{"exec", "--container-id", "test-container", "--workspace-folder", "/workspace", "--override-config", "/cache with spaces/override.json", "/VimRun.sh"}) {
		t.Fatalf("unexpected devcontainer exec args: %#v", args)
	}
}

func TestBuildDevcontainerStartVimExecArgsForShell(t *testing.T) {
	args := buildDevcontainerStartVimExecArgs("test-container", "/workspace", "/cache with spaces/override.json", "bash")

	if !slices.Equal(args, []string{"exec", "--container-id", "test-container", "--workspace-folder", "/workspace", "--override-config", "/cache with spaces/override.json", "bash"}) {
		t.Fatalf("unexpected devcontainer exec args: %#v", args)
	}
}
