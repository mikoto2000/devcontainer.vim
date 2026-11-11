package devcontainer

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigurationParsersPreserveJSONError(t *testing.T) {
	for _, operation := range []struct {
		name  string
		parse func(string) error
	}{
		{"path", func(input string) error { _, err := GetConfigFilePath(input); return err }},
		{"ports", func(input string) error { _, err := GetForwardPorts(input); return err }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			err := operation.parse("{")
			var syntax *json.SyntaxError
			if !errors.As(err, &syntax) {
				t.Fatalf("error = %v, want the original JSON syntax error", err)
			}
		})
	}
}

func TestConfigurationCommandsPreserveProcessFailure(t *testing.T) {
	dir := t.TempDir()
	cli := installStopTestCommands(t, dir, stopTestFixture{ConfigExit: 37, ForbidDocker: true})
	for _, operation := range []struct {
		name string
		run  func() (string, error)
	}{
		{"read", func() (string, error) { return ReadConfiguration(cli, "--workspace-folder", dir) }},
		{"path", func() (string, error) { return GetConfigurationFilePath(cli, dir) }},
		{"create", func() (string, error) { return CreateConfigFile(cli, dir, t.TempDir()) }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			value, err := operation.run()
			var exit *exec.ExitError
			if value != "" || !errors.As(err, &exit) || exit.ExitCode() != 37 ||
				!strings.Contains(err.Error(), "fixture configuration rejected") {
				t.Fatalf("configuration result = %q, %v, want original exit code and stderr", value, err)
			}
		})
	}
}

func TestGetConfigFilePathRejectsMissingPath(t *testing.T) {
	for _, input := range []string{"{}", `{"configuration":{}}`, `{"configuration":{"configFilePath":{"fsPath":""}}}`} {
		if path, err := GetConfigFilePath(input); path != "" || err == nil {
			t.Errorf("GetConfigFilePath(%s) = %q, %v, want a missing-path error", input, path, err)
		}
	}
}

func TestStopAndDownRejectMissingWorkspace(t *testing.T) {
	for _, action := range []struct {
		name string
		run  func([]string, string, string) error
	}{{"stop", Stop}, {"down", Down}} {
		t.Run(action.name, func(t *testing.T) {
			for _, args := range [][]string{nil, {}, {""}} {
				if err := action.run(args, "unused-cli", t.TempDir()); err == nil ||
					!strings.Contains(err.Error(), "missing workspace folder") {
					t.Errorf("arguments %q returned %v, want a workspace error", args, err)
				}
			}
		})
	}
}

func TestStopThenDownIncludesStoppedContainer(t *testing.T) {
	dir := t.TempDir()
	cli := installStopTestCommands(t, dir, stopTestFixture{Workspace: dir, DockerPSOutput: `{"ID":"container-id"}`})
	args := []string{"--nocdr", dir}
	cache := t.TempDir()
	if err := Stop(args, cli, cache); err != nil {
		t.Fatal(err)
	}
	if err := Down(args, cli, cache); err != nil {
		t.Fatalf("Down after Stop failed: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	assertStopTestCall(t, filepath.Join(dir, "finished"), stopTestCall{
		Dir: cwd, Args: []string{"rm", "-f", "container-id"},
	})
}
