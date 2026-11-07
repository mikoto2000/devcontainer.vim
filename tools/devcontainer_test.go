package tools

import (
	"fmt"
	"testing"
)

func TestDevcontainerCLIAssets(t *testing.T) {
	const releaseURL = "https://github.com/mikoto2000/devcontainers-cli/releases/download/v1.2.3/"
	for _, tc := range []struct {
		hostOS, hostArch string
		fileName, asset  string
	}{
		{"darwin", "amd64", "devcontainer", "devcontainer-darwin-x64-v1.2.3"},
		{"darwin", "arm64", "devcontainer", "devcontainer-darwin-arm64-v1.2.3"},
		{"linux", "amd64", "devcontainer", "devcontainer-linux-x64-v1.2.3"},
		{"linux", "arm64", "devcontainer", "devcontainer-linux-arm64-v1.2.3"},
		{"windows", "amd64", "devcontainer.exe", "devcontainer-windows-x64-v1.2.3.exe"},
		{"windows", "arm64", "devcontainer.exe", "devcontainer-windows-x64-v1.2.3.exe"},
	} {
		t.Run(tc.hostOS+"/"+tc.hostArch, func(t *testing.T) {
			tool := devcontainerToolForHost(devcontainerReleaseFixture{}, tc.hostOS, tc.hostArch)
			if tool.FileName != tc.fileName {
				t.Errorf("FileName = %q, want %q", tool.FileName, tc.fileName)
			}
			url, err := tool.CalculateDownloadURL("")
			if want := releaseURL + tc.asset; url != want || err != nil {
				t.Fatalf("CalculateDownloadURL() = %q, %v, want %q, nil", url, err, want)
			}
		})
	}
}

type devcontainerReleaseFixture struct {
	TestInstallerUseServices
}

func (devcontainerReleaseFixture) GetLatestReleaseFromGitHub(owner, repository string) (string, error) {
	if owner != "mikoto2000" || repository != "devcontainers-cli" {
		return "", fmt.Errorf("unexpected release repository %s/%s", owner, repository)
	}
	return "v1.2.3", nil
}
