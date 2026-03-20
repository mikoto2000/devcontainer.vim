package devcontainer

import "testing"

func TestParsePortForwarderMarker(t *testing.T) {
	srcPort, destPort, err := parsePortForwarderMarker("localhost:8080_172.17.0.2:45123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srcPort != "8080" {
		t.Fatalf("expected source port 8080, got %s", srcPort)
	}
	if destPort != "45123" {
		t.Fatalf("expected destination port 45123, got %s", destPort)
	}
}

func TestParsePortForwarderMarkerRejectsInvalidMarker(t *testing.T) {
	_, _, err := parsePortForwarderMarker("localhost:8080")
	if err == nil {
		t.Fatal("expected error for invalid marker")
	}
}
