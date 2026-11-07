package oras

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"oras.land/oras-go/v2/registry/remote/auth"
)

func TestPullCancelsRegistryRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/fixture/manifests/latest" {
			http.NotFound(w, r)
			return
		}
		cancel()
		<-r.Context().Done()
	}))
	defer server.Close()

	client := server.Client()
	client.Timeout = 2 * time.Second
	previous := auth.DefaultClient
	auth.DefaultClient = &auth.Client{Client: client}
	defer func() { auth.DefaultClient = previous }()

	repository := strings.TrimPrefix(server.URL, "https://") + "/fixture"
	if err := Pull(ctx, repository, "latest", t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Pull returned %v after canceling the registry request, want context.Canceled", err)
	}
}
