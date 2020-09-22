package net

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	dtls "github.com/pion/dtls/v2"
)

type connData struct {
	conn net.Conn
	err  error
}

// DTLSListener is a DTLS listener that provides accept with context.
type DTLSListener struct {
	listener  net.Listener
	heartBeat time.Duration
	wg        sync.WaitGroup
	doneCh    chan struct{}
	connCh    chan connData

	closed   uint32
	deadline atomic.Value
}

func (l *DTLSListener) acceptLoop() {
	defer l.wg.Done()
	for {
		conn, err := l.listener.Accept()
		select {
		case l.connCh <- connData{conn: conn, err: err}:
		case <-l.doneCh:
			return
		}
	}
}

var defaultDTLSListenerOptions = dtlsListenerOptions{
	heartBeat: time.Millisecond * 200,
}

type dtlsListenerOptions struct {
	heartBeat time.Duration
}

// A DTLSListenerOption sets options such as heartBeat parameters, etc.
type DTLSListenerOption interface {
	applyDTLSListener(*dtlsListenerOptions)
}

// NewDTLSListener creates dtls listener.
// Known networks are "udp", "udp4" (IPv4-only), "udp6" (IPv6-only).
func NewDTLSListener(network string, addr string, dtlsCfg *dtls.Config, opts ...DTLSListenerOption) (*DTLSListener, error) {
	cfg := defaultDTLSListenerOptions
	for _, o := range opts {
		o.applyDTLSListener(&cfg)
	}

	a, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve address: %w", err)
	}
	listener, err := dtls.Listen(network, a, dtlsCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create new dtls listener: %w", err)
	}
	l := DTLSListener{
		listener:  listener,
		heartBeat: cfg.heartBeat,
		doneCh:    make(chan struct{}),
		connCh:    make(chan connData),
	}
	l.wg.Add(1)

	go l.acceptLoop()

	return &l, nil
}

// AcceptWithContext waits with context for a generic Conn.
func (l *DTLSListener) AcceptWithContext(ctx context.Context) (net.Conn, error) {
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
		rw, err := l.Accept()
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
func (l *DTLSListener) SetDeadline(t time.Time) error {
	l.deadline.Store(t)
	return nil
}

// Accept waits for a generic Conn.
func (l *DTLSListener) Accept() (net.Conn, error) {
	var deadline time.Time
	v := l.deadline.Load()
	if v != nil {
		deadline = v.(time.Time)
	}

	if deadline.IsZero() {
		select {
		case d := <-l.connCh:
			if d.err != nil {
				return nil, d.err
			}
			return d.conn, nil
		}
	}

	select {
	case d := <-l.connCh:
		if d.err != nil {
			return nil, d.err
		}
		return d.conn, nil
	case <-time.After(deadline.Sub(time.Now())):
		return nil, fmt.Errorf(ioTimeout)
	}
}

// Close closes the connection.
func (l *DTLSListener) Close() error {
	if !atomic.CompareAndSwapUint32(&l.closed, 0, 1) {
		return nil
	}
	err := l.listener.Close()
	close(l.doneCh)
	l.wg.Wait()
	return err
}

// Addr represents a network end point address.
func (l *DTLSListener) Addr() net.Addr {
	return l.listener.Addr()
}
