package devcontainer

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

const startWorkflowShellEnv = "DEVCONTAINER_VIM_START_WORKFLOW_SHELL"
const startWorkflowAddressEnv = "DEVCONTAINER_VIM_START_WORKFLOW_ADDRESS"
const startWorkflowEditorArg = "argument with spaces"
const startWorkflowPayload = "data through the host relay"

func TestStartRelaysPortWhileRunningVimLauncher(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executing the generated launcher requires a POSIX shell")
	}
	shell, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("POSIX shell is unavailable")
	}
	t.Setenv("WSL_INTEROP", "")
	t.Setenv("WSL_DISTRO_NAME", "")
	if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
		t.Fatal(err)
	}

	backend := listenForForwardTest(t)
	defer backend.Close()
	go func() {
		conn, err := backend.Accept()
		if err == nil {
			defer conn.Close()
			_, _ = io.WriteString(conn, startWorkflowPayload)
		}
	}()
	portReservation := listenForForwardTest(t)
	hostAddress := portReservation.Addr().String()
	port := portReservation.Addr().(*net.TCPAddr).Port
	defer portReservation.Close()

	cli, fixtureDir := installForwardTestCommands(t, fmt.Sprintf("[%d]", port))
	t.Setenv(startWorkflowShellEnv, shell)
	t.Setenv(startWorkflowAddressEnv, hostAddress)
	t.Setenv("DEVCONTAINER_VIM_FORWARD_TEST_DEST", backend.Addr().String())
	editor, err := os.ReadFile(cli)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "vim"), editor, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixtureDir, "port-forwarder-container_amd64"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	configDir := t.TempDir()
	vimrc := filepath.Join(configDir, "vimrc")
	if err := os.WriteFile(vimrc, nil, 0600); err != nil {
		t.Fatal(err)
	}

	if err := portReservation.Close(); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(configDir, "devcontainer.json")
	if err := Start(DefaultDevcontainerStartUseService{}, []string{fixtureDir}, cli, true, false, true, "", fixtureDir, false, "", configFile, vimrc); err != nil {
		t.Fatal(err)
	}
	assertForwardTestArgs(t, fixtureDir, "editor-args", []string{"--cmd", "let g:devcontainer_vim = v:true", "-S", "/SendToTcp.vim", "-S", "/vimrc", startWorkflowEditorArg})
	portText := fmt.Sprint(port)
	waitForForwardListenerToClose(t, portText)
	assertForwardChildReaped(t, fixtureDir, portText)
}

func runStartEditorFixture(dir string, args []string) error {
	conn, err := net.DialTimeout("tcp", os.Getenv(startWorkflowAddressEnv), 2*time.Second)
	if err != nil {
		return fmt.Errorf("connect from the launched editor: %w", err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		return err
	}
	reply, err := io.ReadAll(conn)
	if err != nil || string(reply) != startWorkflowPayload {
		return fmt.Errorf("editor received %q through the relay, read error %v", reply, err)
	}
	if !writeForwardTestArgs(dir, "editor-args", args) {
		return fmt.Errorf("record editor arguments")
	}
	return nil
}
