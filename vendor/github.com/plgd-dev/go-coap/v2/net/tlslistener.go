package net

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"go.uber.org/atomic"
)

// TLSListener is a TLS listener that provides accept with context.
type TLSListener struct {
	listener net.Listener
	tcp      *net.TCPListener
	closed   atomic.Bool
}

// NewTLSListener creates tcp listener.
// Known networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only).
func NewTLSListener(network string, addr string, tlsCfg *tls.Config) (*TLSListener, error) {
	tcp, err := newNetTCPListen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot create new tls listener: %w", err)
	}
	tls := tls.NewListener(tcp, tlsCfg)
	return &TLSListener{
		tcp:      tcp,
		listener: tls,
	}, nil
}

// AcceptWithContext waits with context for a generic Conn.
func (l *TLSListener) AcceptWithContext(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if l.closed.Load() {
		return nil, ErrListenerIsClosed
	}
	rw, err := l.listener.Accept()
	if err != nil {
		return nil, err
	}
	return rw, nil
}

// Accept waits for a generic Conn.
func (l *TLSListener) Accept() (net.Conn, error) {
	return l.AcceptWithContext(context.Background())
}

// Close closes the connection.
func (l *TLSListener) Close() error {
	if !l.closed.CAS(false, true) {
		return nil
	}
	return l.listener.Close()
}

// Addr represents a network end point address.
func (l *TLSListener) Addr() net.Addr {
	return l.listener.Addr()
}
