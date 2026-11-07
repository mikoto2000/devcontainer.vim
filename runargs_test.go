package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestParseRunargs(t *testing.T) {
	t.Setenv("RUNARGS_TEST_VALUE", "a b; $(printf expanded)")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		input string
		want  []string
	}{
		{
			input: `-e "X=${RUNARGS_TEST_VALUE}" -e '$RUNARGS_TEST_VALUE' --label=two\ words "" "\$RUNARGS_TEST_VALUE"`,
			want:  []string{"-e", "X=a b; $(printf expanded)", "-e", "$RUNARGS_TEST_VALUE", "--label=two words", "", "$RUNARGS_TEST_VALUE"},
		},
		{input: "--init # comment\r\n--label=abc#def \\\n -v ~/vim:/vim *.txt", want: []string{"--init", "--label=abc#def", "-v", home + "/vim:/vim", "*.txt"}},
		{input: `-e '$(touch marker); &' -e "` + "\\`" + `literal"`, want: []string{"-e", "$(touch marker); &", "-e", "`literal"}},
	} {
		got, err := parseRunargs(tc.input)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("parseRunargs(%q) = %q, %v, want %q", tc.input, got, err, tc.want)
		}
	}
	for _, input := range []string{`"unterminated`, "unfinished\\", `${X:-value}`, `$((1+1))`, "x\x00y"} {
		if _, err := parseRunargs(input); err == nil {
			t.Errorf("parseRunargs(%q) succeeded, want error", input)
		}
	}
}

func TestRunargsCLI(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("runargs is used on non-Windows hosts")
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, config, wantErr string
	}{
		{name: "existing defaults", config: runargsContent},
		{name: "command substitution", config: `-e "X=$(touch "$RUNARGS_TEST_MARKER")"`, wantErr: "only $(pwd) is supported"},
		{name: "command separator", config: `--init; touch "$RUNARGS_TEST_MARKER"`, wantErr: "unquoted shell operator"},
		{name: "backticks", config: "-e X=`touch \"$RUNARGS_TEST_MARKER\"`", wantErr: "backtick command substitution"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			home := filepath.Join(dir, "home with spaces")
			workspace := filepath.Join(dir, "work 'with spaces'")
			t.Setenv("HOME", home)
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
			t.Setenv("XDG_CACHE_HOME", filepath.Join(dir, "cache"))
			t.Setenv("WSL_DISTRO_NAME", "")
			if err := os.Unsetenv("WSL_DISTRO_NAME"); err != nil {
				t.Fatal(err)
			}
			t.Setenv(envDevcontainerVimType, "")
			t.Setenv(envDevcontainerShellType, "")
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			marker := filepath.Join(dir, "host-command")
			dockerArgs := filepath.Join(dir, "docker-args")
			t.Setenv("RUNARGS_TEST_MARKER", marker)
			t.Setenv("RUNARGS_TEST_DOCKER_ARGS", dockerArgs)

			configBase, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			cacheBase, err := os.UserCacheDir()
			if err != nil {
				t.Fatal(err)
			}
			configDir := filepath.Join(configBase, appName)
			binDir := filepath.Join(cacheBase, appName, "bin")
			for _, path := range []string{workspace, configDir, binDir} {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(configDir, "runargs"), []byte(tc.config), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(binDir, "clipboard-data-receiver"), nil, 0700); err != nil {
				t.Fatal(err)
			}
			const docker = `#!/bin/sh
case "$1" in
  run) printf '%s\000' "$@" > "$RUNARGS_TEST_DOCKER_ARGS"; printf 'container-id\n' ;;
  exec)
    case "$2:$3" in
      container-id:uname) [ "$4" = -m ] || exit 99; printf 'x86_64\n' ;;
      container-id:sh)
        [ "$4" = -c ] && [ "$6" = devcontainer-vim-tool-probe ] && [ "$7" = vim ] || exit 99
        printf 'found\n' ;;
      --detach-keys:*)
        [ "$4" = -it ] && [ "$5" = container-id ] && [ "$6" = /VimRun.sh ] || exit 99 ;;
      *) exit 99 ;;
    esac ;;
  cp|stop) ;;
  *) exit 99 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(docker), 0700); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(self, "-test.run=^TestCLIProcess$", "--", "--nocdr", "--nopf", "--notmux", "run", "test-image")
			cmd.Dir = workspace
			cmd.Env = append(os.Environ(), "DEVCONTAINER_VIM_TEST_CLI=1")
			out, err := cmd.CombinedOutput()
			if _, statErr := os.Stat(marker); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("runargs executed a host command: %v", statErr)
			}
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(string(out), tc.wantErr) {
					t.Errorf("run error = %v, output = %s, want %q", err, out, tc.wantErr)
				}
				if _, err := os.Stat(dockerArgs); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("Docker was called for invalid runargs: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("run failed: %v\n%s", err, out)
			}
			data, err := os.ReadFile(dockerArgs)
			if err != nil {
				t.Fatal(err)
			}
			got := strings.Split(strings.TrimSuffix(string(data), "\x00"), "\x00")
			// The child resolves the working directory through the OS, which may
			// canonicalize a symlink in the temporary directory path.
			workspace, err = filepath.EvalSymlinks(workspace)
			if err != nil {
				t.Fatal(err)
			}
			want := []string{
				"-v", workspace + ":/work",
				"-v", home + "/.vim:/root/.vim",
				"-v", home + "/.gitconfig:/root/.gitconfig",
				"-v", home + "/.ssh:/root/.ssh",
				"--workdir", "/work", "test-image",
			}
			if image := slices.Index(got, "test-image"); image < len(want)-1 || !slices.Equal(got[image-len(want)+1:image+1], want) {
				t.Errorf("docker run arguments = %q, want sequence %q", got, want)
			}
		})
	}
}
