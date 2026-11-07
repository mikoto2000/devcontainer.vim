package docker

import (
	"errors"
	"strings"
	"testing"
)

func TestGetIDFromDockerPs(t *testing.T) {
	for _, tt := range []struct {
		name, output, id, wantErr string
	}{
		{name: "single", output: " \r\n{\"ID\":\"first\",\"Names\":\"app\"}\r\n\r\n", id: "first"},
		{name: "formatted object", output: "{\n  \"ID\": \"first\"\n}", id: "first"},
		{name: "multiple", output: "{\"ID\":\"first\"}\n{\"ID\":\"second\"}\n", wantErr: `multiple containers match (IDs "first" and "second")`},
		{name: "repeated short ID", output: "{\"ID\":\"first\"}\n{\"ID\":\"first\"}\n", wantErr: "multiple containers match"},
		{name: "malformed second record", output: "{\"ID\":\"first\"}\n{", wantErr: "read docker ps JSON"},
		{name: "missing ID", output: "{}", wantErr: "missing a container ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			id, err := GetID(tt.output)
			if id != tt.id {
				t.Errorf("GetID ID = %q, want %q", id, tt.id)
			}
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("GetID error = %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("GetID error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestGetIDFromEmptyDockerPs(t *testing.T) {
	id, err := GetID(" \r\n")
	var notFound *ContainerNotFoundError
	if id != "" || !errors.As(err, &notFound) {
		t.Fatalf("GetID = %q, %v, want ContainerNotFoundError", id, err)
	}
}
