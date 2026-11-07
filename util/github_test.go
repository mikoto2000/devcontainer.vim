package util

import (
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
		name string
		body string
		tag  string
	}{
		{name: "tag present", body: `{"tag_name":"v1.2.3"}`, tag: "v1.2.3"},
		{name: "tag empty", body: `{"tag_name":""}`},
		{name: "tag missing", body: `{}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/test-owner/test-repo/releases/latest" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
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
			if tt.tag == "" {
				if tag != "" || err == nil || !strings.Contains(err.Error(), "test-owner/test-repo has no tag name") {
					t.Fatalf("getLatestReleaseTag() = %q, %v, want an error for the missing tag", tag, err)
				}
			} else if err != nil || tag != tt.tag {
				t.Fatalf("getLatestReleaseTag() = %q, %v, want %q, nil", tag, err, tt.tag)
			}
		})
	}
}
