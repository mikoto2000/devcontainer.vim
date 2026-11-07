package util

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

var forwardingDialer = net.Dialer{Timeout: 5 * time.Second}

// forward relays traffic between an accepted client and the destination.
func forward(ctx context.Context, src net.Conn, dstAddr string) error {
	dst, err := forwardingDialer.DialContext(ctx, "tcp", dstAddr)
	if err != nil {
		src.Close()
		return err
	}
	closeConnections := func() {
		src.Close()
		dst.Close()
	}
	defer closeConnections()
	stop := context.AfterFunc(ctx, closeConnections)
	defer stop()

	copyDirection := func(to, from net.Conn) error {
		_, err := io.Copy(to, from)
		if writer, ok := to.(interface{ CloseWrite() error }); err == nil && ok {
			err = writer.CloseWrite()
		}
		if err != nil {
			closeConnections()
		}
		return err
	}
	clientDone := make(chan error, 1)
	go func() { clientDone <- copyDirection(dst, src) }()
	serverErr := copyDirection(src, dst)
	return errors.Join(serverErr, <-clientDone)
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
	var retryDelay time.Duration
	for {
		client, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var networkErr net.Error
			if !errors.As(err, &networkErr) || !networkErr.Temporary() {
				return err
			}
			retryDelay = min(time.Second, max(5*time.Millisecond, 2*retryDelay))
			timer := time.NewTimer(retryDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
			continue
		}
		retryDelay = 0
		workers.Go(func() { _ = forward(ctx, client, forwardAddr) })
	}
}
