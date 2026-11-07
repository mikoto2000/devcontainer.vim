package util

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
)

func TestForwardingDialTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dialer := forwardingDialer
		dialer.ControlContext = func(ctx context.Context, _, _ string, _ syscall.RawConn) error {
			// Hold the dial before any connection attempt until it times out.
			<-ctx.Done()
			return ctx.Err()
		}
		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
		defer cancel()
		start := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp4", "127.0.0.1:0")
		if conn != nil {
			conn.Close()
		}
		var timeout net.Error
		if !errors.As(err, &timeout) || !timeout.Timeout() || time.Since(start) != 5*time.Second {
			t.Fatalf("dial returned %v after %s, want a five-second timeout", err, time.Since(start))
		}
	})
}

func TestForwardingLogsConnectionFailure(t *testing.T) {
	messages := make(forwardingLogCapture, 1)
	previous := log.Writer()
	log.SetOutput(messages)
	defer log.SetOutput(previous)

	listener := listenTCPForTest(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	const target = "invalid-target"
	go func() { done <- ServeForwarding(ctx, listener, target) }()

	client, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	select {
	case line := <-messages:
		if !strings.Contains(line, "port forwarding to "+target+" failed:") || !strings.Contains(line, "missing port") {
			t.Errorf("forwarding log = %q, want the destination and dial error", line)
		}
	case <-time.After(2 * time.Second):
		t.Error("forwarder did not log the dial error")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("forwarder did not return after cancellation")
	}
}

type forwardingLogCapture chan string

func (c forwardingLogCapture) Write(p []byte) (int, error) {
	select {
	case c <- string(p):
	default:
	}
	return len(p), nil
}

func TestForwardingStopsOnCancellation(t *testing.T) {
	for _, mode := range []string{"idle", "active"} {
		t.Run(mode, func(t *testing.T) {
			listener := listenTCPForTest(t)
			backend := listenTCPForTest(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- ServeForwarding(ctx, listener, backend.Addr().String()) }()

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

func TestForwardingRelaysHalfCloseInBothDirections(t *testing.T) {
	for _, firstSender := range []string{"client", "server"} {
		t.Run(firstSender, func(t *testing.T) {
			listener := listenTCPForTest(t)
			backend := listenTCPForTest(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- ServeForwarding(ctx, listener, backend.Addr().String()) }()

			conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
			if err != nil {
				t.Fatal(err)
			}
			client := conn.(*net.TCPConn)
			defer client.Close()
			backend.SetDeadline(time.Now().Add(time.Second))
			server, err := backend.AcceptTCP()
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			for _, conn := range []*net.TCPConn{client, server} {
				conn.SetDeadline(time.Now().Add(2 * time.Second))
			}

			sendAndCloseWrite := func(from, to *net.TCPConn, payload string) {
				t.Helper()
				if _, err := io.WriteString(from, payload); err != nil {
					t.Fatal(err)
				}
				if err := from.CloseWrite(); err != nil {
					t.Fatal(err)
				}
				if data, err := io.ReadAll(to); err != nil || string(data) != payload {
					t.Fatalf("received %q, %v; want %q followed by EOF", data, err, payload)
				}
			}
			first, second := client, server
			if firstSender == "server" {
				first, second = server, client
			}
			sendAndCloseWrite(first, second, "request")
			sendAndCloseWrite(second, first, "reply")
			cancel()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("forwarder did not return after both directions finished")
			}
		})
	}
}

func TestForwardingStopsOnPermanentAcceptError(t *testing.T) {
	listener := listenTCPForTest(t)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := ServeForwarding(ctx, listener, "unused"); !errors.Is(err, net.ErrClosed) {
		t.Fatalf("ServeForwarding returned %v, want closed listener error", err)
	}
}

func TestForwardingRetriesTemporaryAcceptError(t *testing.T) {
	listener := &acceptErrorListener{
		Listener: listenTCPForTest(t),
		errors: []error{
			&net.DNSError{Err: "temporary fixture error", IsTemporary: true},
			net.ErrClosed,
		},
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := ServeForwarding(ctx, listener, "unused"); !errors.Is(err, net.ErrClosed) || len(listener.errors) != 0 {
		t.Fatalf("ServeForwarding returned %v with %d errors remaining, want a retry followed by the permanent error", err, len(listener.errors))
	}
}

type acceptErrorListener struct {
	net.Listener
	errors []error
}

func (l *acceptErrorListener) Accept() (net.Conn, error) {
	if len(l.errors) == 0 {
		return nil, net.ErrClosed
	}
	err := l.errors[0]
	l.errors = l.errors[1:]
	return nil, err
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
