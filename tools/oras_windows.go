//go:build windows

package tools

import (
	"os"
	"path/filepath"

	"github.com/mikoto2000/devcontainer.vim/util"
)

const orasFileName = "oras.exe"

// devcontainer-cli のダウンロード URL
const downloadURLOrasPattern = "https://github.com/oras-project/oras/releases/download/{{ .TagName }}/oras_{{ .Version }}_windows_amd64.zip"

func orasInstall(downloadURL string, filePath string) (string, error) {
	// 一時ファイルとしてダウンロード
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, "oras.zip")
	err := download(downloadURL, tempFile)
	if err != nil {
		return filePath, nil
	}

	// 一時出力先組み立て
	tempOrasFile := filepath.Join(tempDir, orasFileName)

	// 展開し、devcontainer.vim の bin ディレクトリへ格納
	err = util.ExtractOneFromZip(tempFile, tempDir, orasFileName)
	if err != nil {
		return filePath, err
	}

	// 本来の出力先へコピー
	err = util.Copy(tempOrasFile, filePath)
	if err != nil {
		return filePath, err
	}

	return filePath, nil
}
