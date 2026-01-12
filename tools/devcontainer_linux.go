//go:build linux

package tools

const devcontainerFileName = "devcontainer"

// devcontainer-cli のダウンロード URL
const downloadURLDevcontainersCliPattern = "https://github.com/mikoto2000/devcontainers-cli/releases/download/{{ .TagName }}/devcontainer-linux-{{ .Arch }}-{{ .TagName }}"
