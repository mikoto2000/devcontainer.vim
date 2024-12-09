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
	ctx := context.Background()
	client := github.NewClient(nil)

	release, _, err := client.Repositories.GetLatestRelease(ctx, owner, repository)
	if err != nil {
		message := fmt.Sprintf("Error getting latest release: %v", err)
		return "", errors.New(message)
	}

	return release.GetTagName(), nil
}
