package util

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestForwardingStopsOnCancellation(t *testing.T) {
	for _, mode := range []string{"idle", "active"} {
		t.Run(mode, func(t *testing.T) {
			listener := listenTCPForTest(t)
			backend := listenTCPForTest(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- serveForwarding(ctx, listener, backend.Addr().String()) }()

			if mode == "active" {
				client, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { client.Close() })
				backend.SetDeadline(time.Now().Add(time.Second))
				server, err := backend.AcceptTCP()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { server.Close() })
				cancel()
				for _, conn := range []net.Conn{client, server} {
					conn.SetReadDeadline(time.Now().Add(time.Second))
					if _, err := conn.Read(make([]byte, 1)); !errors.Is(err, io.EOF) {
						t.Errorf("connection remained open after cancellation: %v", err)
					}
				}
			} else {
				cancel()
			}
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("forwarder did not return after cancellation")
			}
			if conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second); err == nil {
				conn.Close()
				t.Fatal("listener still accepted connections after cancellation")
			}
		})
	}
}

func listenTCPForTest(t *testing.T) *net.TCPListener {
	t.Helper()
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	return listener
}
