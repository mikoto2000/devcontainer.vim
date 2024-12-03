//go:build windows

package tools

const PortForwarderHostFileName = "port-forwarder-host.exe"

// devcontainer-cli のダウンロード URL
const downloadURLPortForwarderCliPattern = "https://github.com/mikoto2000/port-forwarder/releases/download/{{ .TagName }}/port-forwarder-windows-amd64.exe"
