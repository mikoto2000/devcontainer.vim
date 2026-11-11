package util

import (
	"context"
	"net"

	"github.com/mikoto2000/devcontainer.vim/v3/internal/portforward"
)

func StartForwarding(listenAddr, forwardAddr string) error {
	return portforward.StartForwarding(listenAddr, forwardAddr)
}

func StartForwardingContext(ctx context.Context, listenAddr, forwardAddr string) error {
	return portforward.StartForwardingContext(ctx, listenAddr, forwardAddr)
}

func ServeForwarding(ctx context.Context, listener net.Listener, forwardAddr string) error {
	return portforward.ServeForwarding(ctx, listener, forwardAddr)
}
