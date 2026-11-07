package dockercompose

import (
	"errors"
	"testing"
)

func TestComposeCommandErrorMessage(t *testing.T) {
	for _, tc := range []struct {
		command string
		output  string
		want    string
	}{
		{"ps", "", "docker compose ps: exit status 17"},
		{"stop", " \t\n", "docker compose stop: exit status 17"},
		{"down", "  fixture diagnostic\r\n", "docker compose down: exit status 17: fixture diagnostic"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			if got := composeCommandErrorMessage(tc.command, errors.New("exit status 17"), []byte(tc.output)); got != tc.want {
				t.Fatalf("message = %q, want %q", got, tc.want)
			}
		})
	}
}
