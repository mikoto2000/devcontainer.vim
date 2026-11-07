package devcontainer

import (
	"net"
	"syscall"
)

func portForwardListenConfig() net.ListenConfig {
	return net.ListenConfig{Control: func(_, _ string, socket syscall.RawConn) error {
		// Winsock's SO_EXCLUSIVEADDRUSE makes wildcard binds fail when a
		// listener already owns the port on a specific interface.
		const exclusiveAddrUse = ^syscall.SO_REUSEADDR
		var optionErr error
		if err := socket.Control(func(fd uintptr) {
			optionErr = syscall.SetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, exclusiveAddrUse, 1)
		}); err != nil {
			return err
		}
		return optionErr
	}}
}
