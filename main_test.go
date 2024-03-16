package main

import (
	"testing"
)

func TestIsExistsCommandOk(t *testing.T) {
	got := isExistsCommand("ls")
	want := true
	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}
}

func TestIsExistsCommandNg(t *testing.T) {
	got := isExistsCommand("noExistsCommand")
	want := false
	if got != want {
		t.Fatalf("want %v, but %v:", want, got)
	}
}
