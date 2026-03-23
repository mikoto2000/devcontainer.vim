package devcontainer

import "testing"

func TestBuildDockerRunVimExecArgsUsesInteractiveTTYForVimRunScript(t *testing.T) {
	args := buildDockerRunVimExecArgs("test-container", "")

	if len(args) != 6 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "--detach-keys" || args[2] != dockerDetachKeys || args[3] != "-it" || args[4] != "test-container" || args[5] != "/VimRun.sh" {
		t.Fatalf("unexpected docker exec args: %#v", args)
	}
}

func TestBuildDockerRunVimExecArgsUsesInteractiveTTYForShell(t *testing.T) {
	args := buildDockerRunVimExecArgs("test-container", "bash")

	if len(args) != 6 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "--detach-keys" || args[2] != dockerDetachKeys || args[3] != "-it" || args[4] != "test-container" || args[5] != "bash" {
		t.Fatalf("unexpected docker exec args: %#v", args)
	}
}
