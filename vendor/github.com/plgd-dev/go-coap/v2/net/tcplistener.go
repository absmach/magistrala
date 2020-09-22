package net

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// TCPListener is a TCP network listener that provides accept with context.
type TCPListener struct {
	listener  *net.TCPListener
	heartBeat time.Duration
	closed    uint32
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

var defaultTCPListenerOptions = tcpListenerOptions{
	heartBeat: time.Millisecond * 200,
}

type tcpListenerOptions struct {
	heartBeat time.Duration
}

// A TCPListenerOption sets options such as heartBeat parameters, etc.
type TCPListenerOption interface {
	applyTCPListener(*tcpListenerOptions)
}

// NewTCPListener creates tcp listener.
// Known networks are "tcp", "tcp4" (IPv4-only), "tcp6" (IPv6-only).
func NewTCPListener(network string, addr string, opts ...TCPListenerOption) (*TCPListener, error) {
	cfg := defaultTCPListenerOptions
	for _, o := range opts {
		o.applyTCPListener(&cfg)
	}
	tcp, err := newNetTCPListen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot create new tcp listener: %w", err)
	}
	return &TCPListener{listener: tcp, heartBeat: cfg.heartBeat}, nil
}

// AcceptWithContext waits with context for a generic Conn.
func (l *TCPListener) AcceptWithContext(ctx context.Context) (net.Conn, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if atomic.LoadUint32(&l.closed) == 1 {
			return nil, ErrListenerIsClosed
		}
		err := l.SetDeadline(time.Now().Add(l.heartBeat))
		if err != nil {
			return nil, fmt.Errorf("cannot set deadline to accept connection: %w", err)
		}
		rw, err := l.listener.Accept()
		if err != nil {
			// check context in regular intervals and then resume listening
			if isTemporary(err) {
				continue
			}
			return nil, fmt.Errorf("cannot accept connection: %w", err)
		}
		return rw, nil
	}
}

// SetDeadline sets deadline for accept operation.
func (l *TCPListener) SetDeadline(t time.Time) error {
	return l.listener.SetDeadline(t)
}

// Accept waits for a generic Conn.
func (l *TCPListener) Accept() (net.Conn, error) {
	return l.AcceptWithContext(context.Background())
}

// Close closes the connection.
func (l *TCPListener) Close() error {
	if !atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		return nil
	}
	return l.listener.Close()
}

// Addr represents a network end point address.
func (l *TCPListener) Addr() net.Addr {
	return l.listener.Addr()
}
