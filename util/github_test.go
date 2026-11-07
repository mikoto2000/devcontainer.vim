package util

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/go-github/v62/github"
)

func TestGetLatestReleaseTag(t *testing.T) {
	for _, tt := range []struct {
		name   string
		body   string
		tag    string
		status int
	}{
		{name: "tag present", body: `{"tag_name":"v1.2.3"}`, tag: "v1.2.3", status: http.StatusOK},
		{name: "tag empty", body: `{"tag_name":""}`, status: http.StatusOK},
		{name: "tag missing", body: `{}`, status: http.StatusOK},
		{name: "release not found", body: `{"message":"Not Found"}`, status: http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/test-owner/test-repo/releases/latest" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()
			client := github.NewClient(server.Client())
			baseURL, err := url.Parse(server.URL + "/")
			if err != nil {
				t.Fatal(err)
			}
			client.BaseURL = baseURL

			tag, err := getLatestReleaseTag(client, "test-owner", "test-repo")
			if tt.status != http.StatusOK {
				var responseErr *github.ErrorResponse
				if tag != "" || !errors.As(err, &responseErr) || responseErr.Response.StatusCode != tt.status {
					t.Fatalf("getLatestReleaseTag() = %q, %v, want the GitHub HTTP %d error", tag, err, tt.status)
				}
			} else if tt.tag == "" {
				if tag != "" || err == nil || !strings.Contains(err.Error(), "test-owner/test-repo has no tag name") {
					t.Fatalf("getLatestReleaseTag() = %q, %v, want an error for the missing tag", tag, err)
				}
			} else if err != nil || tag != tt.tag {
				t.Fatalf("getLatestReleaseTag() = %q, %v, want %q, nil", tag, err, tt.tag)
			}
		})
	}
}
