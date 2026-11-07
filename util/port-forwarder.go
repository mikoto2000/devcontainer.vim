package util

import (
	"context"
	"io"
	"net"
	"sync"
)

// forward relays traffic between an accepted client and the destination.
func forward(ctx context.Context, src net.Conn, dstAddr string) error {
	var dialer net.Dialer
	dst, err := dialer.DialContext(ctx, "tcp", dstAddr)
	if err != nil {
		src.Close()
		return err
	}
	closeConnections := func() {
		src.Close()
		dst.Close()
	}
	stop := context.AfterFunc(ctx, closeConnections)
	defer stop()

	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		io.Copy(dst, src)
	}()
	defer func() {
		closeConnections()
		<-clientDone
	}()
	io.Copy(src, dst)
	return nil
}

// StartForwarding relays TCP connections without a cancellation context.
func StartForwarding(listenAddr, forwardAddr string) error {
	return StartForwardingContext(context.Background(), listenAddr, forwardAddr)
}

// StartForwardingContext relays TCP connections until ctx is canceled.
func StartForwardingContext(ctx context.Context, listenAddr, forwardAddr string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var config net.ListenConfig
	listener, err := config.Listen(ctx, "tcp", listenAddr)
	if err != nil {
		return err
	}
	return serveForwarding(ctx, listener, forwardAddr)
}

func serveForwarding(ctx context.Context, listener net.Listener, forwardAddr string) error {
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	stop := context.AfterFunc(ctx, func() { listener.Close() })
	defer func() {
		cancel()
		stop()
		listener.Close()
		workers.Wait()
	}()
	for {
		client, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			continue
		}
		workers.Go(func() { _ = forward(ctx, client, forwardAddr) })
	}
}
