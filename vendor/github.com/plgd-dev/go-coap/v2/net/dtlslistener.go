package net

import (
	"context"
	"fmt"
	"net"
	"time"

	dtls "github.com/pion/dtls/v2"
	"go.uber.org/atomic"
)

// DTLSListener is a DTLS listener that provides accept with context.
type DTLSListener struct {
	listener net.Listener
	closed   atomic.Bool
}

// NewDTLSListener creates dtls listener.
// Known networks are "udp", "udp4" (IPv4-only), "udp6" (IPv6-only).
func NewDTLSListener(network string, addr string, dtlsCfg *dtls.Config) (*DTLSListener, error) {
	a, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve address: %w", err)
	}

	var l DTLSListener
	connectContextMaker := dtlsCfg.ConnectContextMaker
	if connectContextMaker == nil {
		connectContextMaker = func() (context.Context, func()) {
			return context.WithTimeout(context.Background(), 30*time.Second)
		}
	}
	dtlsCfg.ConnectContextMaker = func() (context.Context, func()) {
		ctx, cancel := connectContextMaker()
		if l.closed.Load() {
			cancel()
		}
		return ctx, cancel
	}

	listener, err := dtls.Listen(network, a, dtlsCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create new dtls listener: %w", err)
	}
	l.listener = listener
	return &l, nil
}

// AcceptWithContext waits with context for a generic Conn.
func (l *DTLSListener) AcceptWithContext(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if l.closed.Load() {
		return nil, ErrListenerIsClosed
	}
	c, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, nil
	}
	return c, nil
}

// Accept waits for a generic Conn.
func (l *DTLSListener) Accept() (net.Conn, error) {
	return l.AcceptWithContext(context.Background())
}

// Close closes the connection.
func (l *DTLSListener) Close() error {
	if !l.closed.CAS(false, true) {
		return nil
	}
	return l.listener.Close()
}

// Addr represents a network end point address.
func (l *DTLSListener) Addr() net.Addr {
	return l.listener.Addr()
}
