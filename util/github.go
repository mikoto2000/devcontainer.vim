package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v62/github"
)

/**
 * ユーザー名、リポジトリ名から最新リリースタグ名を返却する。
 */
func GetLatestReleaseFromGitHub(owner string, repository string) (string, error) {
	return getLatestReleaseTag(github.NewClient(nil), owner, repository)
}

func getLatestReleaseTag(client *github.Client, owner, repository string) (string, error) {
	ctx := context.Background()

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repository)
	if err != nil {
		message := fmt.Sprintf("Error getting latest release: %v", err)
		return "", errors.New(message)
	}

	tag := release.GetTagName()
	if tag == "" {
		return "", fmt.Errorf("latest release for %s/%s has no tag name", owner, repository)
	}
	return tag, nil
}
