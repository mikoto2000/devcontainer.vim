//go:build !windows

package tools

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mikoto2000/devcontainer.vim/util"
)

const orasFileName = "oras"

// devcontainer-cli のダウンロード URL
const downloadURLOrasPattern = "https://github.com/oras-project/oras/releases/download/{{ .TagName }}/oras_{{ .Version }}_linux_amd64.tar.gz"

func orasInstall(downloadURL string, filePath string) (string, error) {
	// 一時ファイルとしてダウンロード
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "oras.tar.gz")
	err := download(downloadURL, tempFile)
	if err != nil {
		return filePath, nil
	}

	// 一時出力先組み立て
	tempOrasFile := filepath.Join(tempDir, orasFileName)

	// 展開し、devcontainer.vim の bin ディレクトリへ格納
	fmt.Printf("Extracting %s to %s ... ", tempFile, tempDir)
	err = util.ExtractOneFromTgz(tempFile, tempDir, orasFileName)
	if err != nil {
		return filePath, err
	}
	fmt.Printf("done.\n")

	// 本来の出力先へコピー
	fmt.Printf("Coping %s -> %s ... ", tempOrasFile, filePath)
	err = util.Copy(tempOrasFile, filePath)
	if err != nil {
		return filePath, err
	}
	fmt.Printf("done.\n")

	return filePath, nil
}
