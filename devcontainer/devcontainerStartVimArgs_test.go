package devcontainer

import "testing"

func TestBuildDevcontainerStartVimExecArgsUsesDetachKeysForVimRunScript(t *testing.T) {
	args := buildDevcontainerStartVimExecArgs("test-container", "/workspace", "")

	if len(args) != 6 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "--container-id" || args[2] != "test-container" || args[3] != "--workspace-folder" || args[4] != "/workspace" || args[5] != "/VimRun.sh" {
		t.Fatalf("unexpected devcontainer exec args: %#v", args)
	}
}

func TestBuildDevcontainerStartVimExecArgsUsesDetachKeysForShell(t *testing.T) {
	args := buildDevcontainerStartVimExecArgs("test-container", "/workspace", "bash")

	if len(args) != 6 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "--container-id" || args[2] != "test-container" || args[3] != "--workspace-folder" || args[4] != "/workspace" || args[5] != "bash" {
		t.Fatalf("unexpected devcontainer exec args: %#v", args)
	}
}
