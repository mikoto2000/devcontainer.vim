package devcontainer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const forwardTestDirEnv = "DEVCONTAINER_VIM_FORWARD_TEST_DIR"

func TestGetForwardPortsValid(t *testing.T) {
	input := `{"configuration":{"forwardPorts":[0,8080,65535,"api:443","db.local:00543","127.0.0.1:80"]}}`
	want := []ForwardConfig{
		{Host: "localhost", Port: "0"}, {Host: "localhost", Port: "8080"},
		{Host: "localhost", Port: "65535"}, {Host: "api", Port: "443"},
		{Host: "db.local", Port: "543"}, {Host: "127.0.0.1", Port: "80"},
	}
	got, err := GetForwardPorts(input)
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("GetForwardPorts = %v, %v, want %v", got, err, want)
	}
}

func TestGetForwardPortsInvalid(t *testing.T) {
	for _, entry := range []string{
		`"8080"`, `":8080"`, `"api:"`, `"api:80:81"`, `"api:+80"`, `"api:65536"`,
		`"api:80; echo injected"`, `"$(echo injected):80"`, `"api/other:80"`,
		`-1`, `65536`, `8080.5`, `65535.00000000000001`, `true`, `null`, `{}`, `[]`,
	} {
		t.Run(entry, func(t *testing.T) {
			input := `{"configuration":{"forwardPorts":[8080,` + entry + `]}}`
			got, err := GetForwardPorts(input)
			if err == nil || !strings.Contains(err.Error(), "forwardPorts[1]") || len(got) != 0 {
				t.Fatalf("GetForwardPorts = %v, %v, want indexed error and no ports", got, err)
			}
		})
	}
	for _, value := range []string{`"api:80"`, `{}`, `null`} {
		got, err := GetForwardPorts(`{"configuration":{"forwardPorts":` + value + `}}`)
		if err == nil || !strings.Contains(err.Error(), "forwardPorts") || len(got) != 0 {
			t.Errorf("forwardPorts %s = %v, %v, want array error", value, got, err)
		}
	}
}

func TestSetupPortForwardingRejectsInvalidBeforeRunningCommands(t *testing.T) {
	devcontainerPath, dir := installForwardTestCommands(t, `[8080,"localhost:80; echo injected"]`)
	err := setupPortForwarding(t.Context(), "container-id", devcontainerPath, "workspace", "override.json")
	if err == nil || !strings.Contains(err.Error(), "forwardPorts[1]") {
		t.Fatalf("setupPortForwarding error = %v, want validation error", err)
	}
	for _, name := range []string{"docker-called", "exec-called"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("unexpected forwarding command %s: %v", name, err)
		}
	}
}

func TestSetupPortForwardingAcceptsNumberAndHostPort(t *testing.T) {
	// Isolate the fake Docker and Dev Containers subprocesses.
	const childEnv = "DEVCONTAINER_VIM_FORWARD_TEST_CHILD"
	if os.Getenv(childEnv) == "" {
		self, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.CommandContext(t.Context(), self, "-test.run=^TestSetupPortForwardingAcceptsNumberAndHostPort$", "-test.timeout=15s")
		cmd.Env = append(os.Environ(), childEnv+"=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("forwarding test subprocess: %v\n%s", err, output)
		}
		return
	}

	destination := listenForForwardTest(t)
	defer destination.Close()
	go func() {
		for {
			conn, err := destination.Accept()
			if err != nil {
				return
			}
			io.WriteString(conn, "ok")
			conn.Close()
		}
	}()
	first, second := listenForForwardTest(t), listenForForwardTest(t)
	_, firstPort, _ := net.SplitHostPort(first.Addr().String())
	_, secondPort, _ := net.SplitHostPort(second.Addr().String())
	first.Close()
	second.Close()

	devcontainerPath, dir := installForwardTestCommands(t, fmt.Sprintf(`[%s,"api:%s"]`, firstPort, secondPort))
	t.Setenv("DEVCONTAINER_VIM_FORWARD_TEST_DEST", destination.Addr().String())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	if err := setupPortForwarding(ctx, "container-id", devcontainerPath, "workspace", "override.json"); err != nil {
		t.Fatal(err)
	}
	for _, target := range []ForwardConfig{{Host: "localhost", Port: firstPort}, {Host: "api", Port: secondPort}} {
		addr := net.JoinHostPort("127.0.0.1", target.Port)
		deadline := time.Now().Add(3 * time.Second)
		for {
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err == nil {
				conn.SetReadDeadline(deadline)
				reply := make([]byte, 2)
				_, err = io.ReadFull(conn, reply)
				conn.Close()
				if err != nil || string(reply) != "ok" {
					t.Fatalf("forwarded reply for %s = %q, %v", addr, reply, err)
				}
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("forwarding did not start for %s: %v", addr, err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		targetAddr := net.JoinHostPort(target.Host, target.Port)
		assertForwardTestArgs(t, dir, "exec-"+target.Port, []string{"exec", "--container-id", "container-id", "--workspace-folder", "workspace",
			"--override-config", "override.json", "/port-forwarder", "-l", "0.0.0.0:0", "-f", targetAddr})
		assertForwardTestArgs(t, dir, "marker-"+target.Port, []string{"exec", "-t", "container-id", "sh", "-c",
			`dir="$HOME/.config/devcontainer.vim/pf"; mkdir -p "$dir" && touch "$dir/$1"`, "port-forwarder-marker", targetAddr + "_" + destination.Addr().String()})
	}
	cancel()
	for _, port := range []string{firstPort, secondPort} {
		addr := net.JoinHostPort("127.0.0.1", port)
		deadline := time.Now().Add(time.Second)
		for {
			conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
			if err != nil {
				break
			}
			conn.Close()
			if time.Now().After(deadline) {
				t.Fatalf("forwarding for %s continued after cancellation", addr)
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func listenForForwardTest(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	return listener
}

func installForwardTestCommands(t *testing.T, ports string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	devcontainerPath := filepath.Join(dir, "test-forward-devcontainer"+suffix)
	for _, path := range []string{filepath.Join(dir, "docker"+suffix), devcontainerPath} {
		if err := os.WriteFile(path, binary, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(forwardTestDirEnv, dir)
	t.Setenv("DEVCONTAINER_VIM_FORWARD_TEST_PORTS", ports)
	return devcontainerPath, dir
}

func assertForwardTestArgs(t *testing.T, dir, name string, want []string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Errorf("%s args = %q, want %q", name, got, want)
	}
}

func runForwardTestCommand(dir string, argv []string) int {
	args := argv[1:]
	if strings.TrimSuffix(filepath.Base(argv[0]), ".exe") == "test-forward-devcontainer" {
		if args[0] == "read-configuration" {
			fmt.Printf(`{"configuration":{"forwardPorts":%s}}`, os.Getenv("DEVCONTAINER_VIM_FORWARD_TEST_PORTS"))
			return 0
		}
		if err := os.WriteFile(filepath.Join(dir, "exec-called"), nil, 0600); err != nil {
			return 99
		}
		if len(args) != 12 || args[7] != "/port-forwarder" {
			return 99
		}
		_, port, err := net.SplitHostPort(args[11])
		if err != nil || !writeForwardTestArgs(dir, "exec-"+port, args) {
			return 99
		}
		_, dest, err := net.SplitHostPort(os.Getenv("DEVCONTAINER_VIM_FORWARD_TEST_DEST"))
		if err != nil {
			return 99
		}
		fmt.Println("45123; echo injected")
		fmt.Println(dest)
		return 0
	}
	if err := os.WriteFile(filepath.Join(dir, "docker-called"), nil, 0600); err != nil {
		return 99
	}
	if len(args) >= 4 && args[3] == "hostname" {
		fmt.Println("127.0.0.1")
	} else if len(args) == 8 {
		target, _, _ := strings.Cut(args[7], "_")
		_, port, err := net.SplitHostPort(target)
		if err != nil || !writeForwardTestArgs(dir, "marker-"+port, args) {
			return 99
		}
	}
	return 0
}

func writeForwardTestArgs(dir, name string, args []string) bool {
	data, err := json.Marshal(args)
	return err == nil && os.WriteFile(filepath.Join(dir, name), data, 0600) == nil
}
