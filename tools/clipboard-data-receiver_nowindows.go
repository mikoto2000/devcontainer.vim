//go:build !windows

package tools

const CDR_FILE_NAME = "clipboard-data-receiver"

// clipboard-data-receiver のダウンロード URL
const DOWNLOAD_URL_CDR_PATTERN = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.linux-amd64"


func RunCdr(cdrFilePath string, configFileDir string) (int, int, error) {
	panic("not implement.")
}
