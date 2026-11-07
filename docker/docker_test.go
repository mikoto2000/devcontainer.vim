package docker

import (
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

func requireDockerPsJSONSupport(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker command not found, skipping test")
	}

	if err := exec.Command("docker", "ps", "--format", "json").Run(); err != nil {
		t.Skipf("docker ps --format json not available in this environment: %v", err)
	}
}

func assertPsJSON(t *testing.T, output, wantState string) {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(output))
	for {
		var row struct {
			PsCommandResult
			State string
		}
		err := decoder.Decode(&row)
		if err == io.EOF {
			return
		}
		if err != nil {
			t.Fatalf("invalid docker ps JSON: %v\n%s", err, output)
		}
		if strings.TrimSpace(row.ID) == "" {
			t.Fatalf("docker ps row has no ID: %+v", row)
		}
		if wantState != "" && row.State != wantState {
			t.Errorf("container %s has state %q, want %q", row.ID, row.State, wantState)
		}
	}
}

func TestPs(t *testing.T) {
	requireDockerPsJSONSupport(t)

	result, err := Ps("status=running")
	if err != nil {
		t.Fatalf("Ps failed: %v", err)
	}
	assertPsJSON(t, result, "running")
}

func TestPsWithInvalidFilter(t *testing.T) {
	requireDockerPsJSONSupport(t)

	result, err := Ps("invalid=filter")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || !strings.Contains(strings.ToLower(string(exit.Stderr)), "invalid filter") {
		t.Fatalf("Ps with invalid filter returned %q, %v, want an invalid filter error", result, err)
	}
	if strings.TrimSpace(result) != "" {
		t.Errorf("Ps with invalid filter returned container data: %q", result)
	}
}

func TestPsWithEmptyFilter(t *testing.T) {
	requireDockerPsJSONSupport(t)

	result, err := Ps("")
	if err != nil {
		t.Fatalf("Ps with empty filter failed: %v", err)
	}
	assertPsJSON(t, result, "")
}

func TestPsWithValidLabel(t *testing.T) {
	requireDockerPsJSONSupport(t)

	result, err := Ps("label=devcontainer.local_folder=" + t.TempDir())
	if err != nil {
		t.Fatalf("Ps with label filter failed: %v", err)
	}
	if strings.TrimSpace(result) != "" {
		t.Fatalf("Ps with nonexistent label returned containers: %q", result)
	}
}
