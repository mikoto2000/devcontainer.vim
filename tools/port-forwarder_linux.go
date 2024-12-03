//go:build linux

package tools

const PortForwarderHostFileName = "port-forwarder-host"

// devcontainer-cli のダウンロード URL
const downloadURLPortForwarderCliPattern = "https://github.com/mikoto2000/port-forwarder/releases/download/{{ .TagName }}/port-forwarder-linux-amd64"
