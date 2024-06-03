//go:build windows

package tools

const DEVCONTAINER_FILE_NAME = "devcontainer.exe"

// devcontainer-cli のダウンロード URL
const DOWNLOAD_URL_DEVCONTAINERS_CLI_PATTERN = "https://github.com/mikoto2000/devcontainers-cli/releases/download/{{ .TagName }}/devcontainer-windows-x64-{{ .TagName }}.exe"
