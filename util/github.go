package util

import "github.com/mikoto2000/devcontainer.vim/v3/internal/githubrelease"

func GetLatestReleaseFromGitHub(owner, repository string) (string, error) {
	return githubrelease.GetLatestReleaseFromGitHub(owner, repository)
}
