package util

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v62/github"
)

type GetLatestReleaseFromGitHubService interface {
	GetLatestReleaseFromGitHub(owner string, repo string) (string, error)
}

type DefaultGetLatestReleaseFromGitHubService struct{}

func (s DefaultGetLatestReleaseFromGitHubService) GetLatestReleaseFromGitHub(owner string, repo string) (string, error) {
	return GetLatestReleaseFromGitHub(owner, repo)
}

var GetGetLatestReleaseFromGitHubService = func() GetLatestReleaseFromGitHubService {
	dglrfg := DefaultGetLatestReleaseFromGitHubService{}
	return dglrfg
}

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
