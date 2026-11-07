package util

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v62/github"
)

/**
 * ユーザー名、リポジトリ名から最新リリースタグ名を返却する。
 */
func GetLatestReleaseFromGitHub(owner string, repository string) (string, error) {
	var httpClient *http.Client
	if token := os.Getenv("DEVCONTAINER_VIM_GITHUB_TOKEN"); token != "" {
		httpClient = &http.Client{Transport: releaseAuthTransport{base: http.DefaultTransport, token: token}}
	}
	return getLatestReleaseTag(github.NewClient(httpClient), owner, repository)
}

type releaseAuthTransport struct {
	base  http.RoundTripper
	token string
}

func (t releaseAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Apply authentication only to the GitHub API, including after redirects.
	if req.URL.Scheme == "https" && req.URL.Host == "api.github.com" {
		req = req.Clone(req.Context())
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	return t.base.RoundTrip(req)
}

func getLatestReleaseTag(client *github.Client, owner, repository string) (string, error) {
	ctx := context.Background()

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repository)
	if err != nil {
		return "", fmt.Errorf("Error getting latest release: %w", err)
	}

	tag := release.GetTagName()
	if tag == "" {
		return "", fmt.Errorf("latest release for %s/%s has no tag name", owner, repository)
	}
	return tag, nil
}
