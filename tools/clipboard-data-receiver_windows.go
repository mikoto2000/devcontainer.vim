//go:build windows

package tools

const CDR_FILE_NAME = "clipboard-data-receiver.exe"

// clipboard-data-receiver のダウンロード URL
const DOWNLOAD_URL_CDR_PATTERN = "https://github.com/mikoto2000/clipboard-data-receiver/releases/download/{{ .TagName }}/clipboard-data-receiver.windows-amd64.exe"

