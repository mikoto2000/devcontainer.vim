package dockercompose

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mikoto2000/devcontainer.vim/v3/docker"
)

type PsCommandError struct {
	msg string
	err error
}

func (e *PsCommandError) Error() string { return e.msg }
func (e *PsCommandError) Unwrap() error { return e.err }

type StopCommandError struct {
	msg string
	err error
}

func (e *StopCommandError) Error() string { return e.msg }
func (e *StopCommandError) Unwrap() error { return e.err }

type DownCommandError struct {
	msg string
	err error
}

func (e *DownCommandError) Error() string { return e.msg }
func (e *DownCommandError) Unwrap() error { return e.err }

type Project struct {
	Name            string
	workspaceFolder string
	files           []string
}

// FindProject uses labels on existing Dev Containers, including stopped containers.
func FindProject(workspaceFolder, configFile string, files ...string) (Project, error) {
	workspaceFolder, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return Project{}, err
	}
	const format = `{"workspace":{{json (.Label "devcontainer.local_folder")}},"config":{{json (.Label "devcontainer.config_file")}},"project":{{json (.Label "com.docker.compose.project")}}}`
	cmd := exec.Command(docker.Command, "ps", "--all", "--filter", "label=devcontainer.local_folder", "--format", format)
	output, err := cmd.Output()
	if err != nil {
		return Project{}, fmt.Errorf("find Compose project: %w", err)
	}
	project := Project{workspaceFolder: workspaceFolder, files: files}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var labels struct {
			Workspace string
			Config    string
			Project   string
		}
		if err := json.Unmarshal([]byte(line), &labels); err != nil {
			return Project{}, err
		}
		if labels.Project == "" || !sameHostPath(labels.Workspace, workspaceFolder) {
			continue
		}
		if labels.Config != "" && configFile != "" && !sameHostPath(labels.Config, configFile) {
			continue
		}
		if project.Name != "" && project.Name != labels.Project {
			return Project{}, fmt.Errorf("multiple Compose projects match workspace %q", workspaceFolder)
		}
		project.Name = labels.Project
	}
	if project.Name != "" {
		return project, nil
	}

	// A Compose project started outside Dev Containers may not have its labels.
	result, err := project.ps()
	if err != nil {
		return Project{}, err
	}
	project.Name, err = GetProjectName(result)
	return project, err
}

func sameHostPath(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

// Ps returns the JSON output of docker compose ps.
func Ps(workspaceFolder string) (string, error) {
	return (Project{workspaceFolder: workspaceFolder}).ps()
}

func (p Project) ps() (string, error) {
	cmd := p.command("ps", "--all", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		var diagnostic []byte
		if exit, ok := err.(*exec.ExitError); ok {
			diagnostic = exit.Stderr
		}
		return "", &PsCommandError{msg: composeCommandErrorMessage("ps", err, diagnostic), err: err}
	}
	return string(output), nil
}

// Stop waits for docker compose stop to finish.
func Stop(projectName string) error {
	return (Project{Name: projectName}).Stop()
}

func (p Project) Stop() error {
	output, err := p.command("-p", p.Name, "stop").CombinedOutput()
	if err != nil {
		return &StopCommandError{msg: composeCommandErrorMessage("stop", err, output), err: err}
	}
	return nil
}

// Down waits for docker compose down to finish.
func Down(projectName string) error {
	return (Project{Name: projectName}).Down()
}

func (p Project) Down() error {
	output, err := p.command("-p", p.Name, "down").CombinedOutput()
	if err != nil {
		return &DownCommandError{msg: composeCommandErrorMessage("down", err, output), err: err}
	}
	return nil
}

func composeCommandErrorMessage(command string, err error, output []byte) string {
	message := fmt.Sprintf("docker compose %s: %v", command, err)
	if detail := strings.TrimSpace(string(output)); detail != "" {
		message += ": " + detail
	}
	return message
}

func (p Project) command(args ...string) *exec.Cmd {
	commandArgs := []string{"compose"}
	for _, file := range p.files {
		commandArgs = append(commandArgs, "-f", file)
	}
	cmd := exec.Command(docker.Command, append(commandArgs, args...)...)
	// Match Dev Containers' working directory for workspace .env files.
	cmd.Dir = p.workspaceFolder
	return cmd
}
