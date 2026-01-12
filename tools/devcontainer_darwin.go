//go:build darwin

package tools

const devcontainerFileName = "devcontainer"

// devcontainer-cli のダウンロード URL
const downloadURLDevcontainersCliPattern = "https://github.com/mikoto2000/devcontainers-cli/releases/download/{{ .TagName }}/devcontainer-darwin-{{ .Arch }}-{{ .TagName }}"
