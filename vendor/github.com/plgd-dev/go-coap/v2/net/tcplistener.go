package net

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/atomic"
)

// TCPListener is a TCP network listener that provides accept with context.
type TCPListener struct {
	listener *net.TCPListener
	closed   atomic.Bool
}

func newNetTCPListen(network string, addr string) (*net.TCPListener, error) {
	a, err := net.ResolveTCPAddr(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot create new net tcp listener: %w", err)
	}

	tcp, err := net.ListenTCP(network, a)
	if err != nil {
		return nil, fmt.Errorf("cannot create new net tcp listener: %w", err)
	}
	return tcp, nil
}

// NewTCPListener creates tcp listener.
// Known networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only).
func NewTCPListener(network string, addr string) (*TCPListener, error) {
	tcp, err := newNetTCPListen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot create new tcp listener: %w", err)
	}
	return &TCPListener{listener: tcp}, nil
}

// AcceptWithContext waits with context for a generic Conn.
func (l *TCPListener) AcceptWithContext(ctx context.Context) (net.Conn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if l.closed.Load() {
		return nil, ErrListenerIsClosed
	}
	return l.listener.Accept()
}

// Accept waits for a generic Conn.
func (l *TCPListener) Accept() (net.Conn, error) {
	return l.AcceptWithContext(context.Background())
}

// Close closes the connection.
func (l *TCPListener) Close() error {
	if !l.closed.CAS(false, true) {
		return nil
	}
	return l.listener.Close()
}

// Addr represents a network end point address.
func (l *TCPListener) Addr() net.Addr {
	return l.listener.Addr()
}
