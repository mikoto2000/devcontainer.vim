//go:build windows

package tools

const devcontainerFileName = "devcontainer.exe"

// devcontainer-cli のダウンロード URL
const downloadURLDevcontainersCliPattern = "https://github.com/mikoto2000/devcontainers-cli/releases/download/{{ .TagName }}/devcontainer-windows-x64-{{ .TagName }}.exe"
