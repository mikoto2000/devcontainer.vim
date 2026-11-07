//go:build !windows

package devcontainer

import "net"

func portForwardListenConfig() net.ListenConfig {
	return net.ListenConfig{}
}
