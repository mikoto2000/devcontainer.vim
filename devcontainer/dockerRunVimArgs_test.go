package devcontainer

import "testing"

func TestBuildDockerRunVimExecArgsUsesInteractiveTTYForVimRunScript(t *testing.T) {
	args := buildDockerRunVimExecArgs("test-container", "")

	if len(args) != 4 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "-it" || args[2] != "test-container" || args[3] != "/VimRun.sh" {
		t.Fatalf("unexpected docker exec args: %#v", args)
	}
}

func TestBuildDockerRunVimExecArgsUsesInteractiveTTYForShell(t *testing.T) {
	args := buildDockerRunVimExecArgs("test-container", "bash")

	if len(args) != 4 {
		t.Fatalf("unexpected args length: %#v", args)
	}
	if args[0] != "exec" || args[1] != "-it" || args[2] != "test-container" || args[3] != "bash" {
		t.Fatalf("unexpected docker exec args: %#v", args)
	}
}
