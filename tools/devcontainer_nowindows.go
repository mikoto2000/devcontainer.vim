//go:build !windows

package tools

const DEVCONTAINER_FILE_NAME = "devcontainer"

// devcontainer-cli のダウンロード URL
// ※ 全ての `%s` はリリースタグ名
const DOWNLOAD_URL_DEVCONTAINERS_CLI_PATTERN = "https://github.com/mikoto2000/devcontainers-cli/releases/download/%s/devcontainer-linux-x64-%s"
