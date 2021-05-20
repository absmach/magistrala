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
	onTimeout func() error

	cancel context.CancelFunc
	mutex  sync.Mutex

	closed   uint32
	deadline atomic.Value
}

func (l *DTLSListener) acceptLoop() {
	defer l.wg.Done()
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			select {
			case <-l.doneCh:
				return
			case l.connCh <- connData{conn: conn, err: err}:
			}
		} else {
			select {
			case l.connCh <- connData{conn: conn, err: err}:
			case <-l.doneCh:
				return
			}
		}
	}
}

var defaultDTLSListenerOptions = dtlsListenerOptions{
	heartBeat: time.Millisecond * 200,
}

type dtlsListenerOptions struct {
	heartBeat time.Duration
	onTimeout func() error
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
	l := DTLSListener{
		heartBeat: cfg.heartBeat,
		connCh:    make(chan connData),
		doneCh:    make(chan struct{}),
	}

	connectContextMaker := dtlsCfg.ConnectContextMaker
	if connectContextMaker == nil {
		connectContextMaker = func() (context.Context, func()) {
			return context.WithTimeout(context.Background(), 30*time.Second)
		}
	}
	dtlsCfg.ConnectContextMaker = func() (context.Context, func()) {
		ctx, cancel := connectContextMaker()
		l.mutex.Lock()
		defer l.mutex.Unlock()
		if l.closed > 0 {
			cancel()
		}
		l.cancel = cancel
		return ctx, cancel
	}

	listener, err := dtls.Listen(network, a, dtlsCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot create new dtls listener: %w", err)
	}
	l.listener = listener
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
		deadline := time.Now().Add(l.heartBeat)
		err := l.SetDeadline(deadline)
		if err != nil {
			return nil, fmt.Errorf("cannot set deadline to accept connection: %w", err)
		}
		rw, err := l.Accept()
		if err != nil {
			// check context in regular intervals and then resume listening
			if isTemporary(err, deadline) {
				if l.onTimeout != nil {
					err := l.onTimeout()
					if err != nil {
						return nil, fmt.Errorf("cannot accept connection : on timeout returns error: %w", err)
					}
				}
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

func (l *DTLSListener) close() (bool, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.closed > 0 {
		return false, nil
	}
	close(l.doneCh)
	err := l.listener.Close()
	if l.cancel != nil {
		l.cancel()
	}
	return true, err
}

// Close closes the connection.
func (l *DTLSListener) Close() error {
	wait, err := l.close()
	if wait {
		l.wg.Wait()
	}
	return err
}

// Addr represents a network end point address.
func (l *DTLSListener) Addr() net.Addr {
	return l.listener.Addr()
}
