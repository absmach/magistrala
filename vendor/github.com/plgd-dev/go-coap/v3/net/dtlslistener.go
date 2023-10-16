package net

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	dtls "github.com/pion/dtls/v2"
	dtlsnet "github.com/pion/dtls/v2/pkg/net"
	"github.com/pion/dtls/v2/pkg/protocol"
	"github.com/pion/dtls/v2/pkg/protocol/recordlayer"
	"github.com/pion/transport/v3/udp"
	"go.uber.org/atomic"
)

type GoPoolFunc = func(f func()) error

var DefaultDTLSListenerConfig = DTLSListenerConfig{
	GoPool: func(f func()) error {
		go f()
		return nil
	},
}

type DTLSListenerConfig struct {
	GoPool GoPoolFunc
}

type acceptedConn struct {
	conn net.Conn
	err  error
}

// DTLSListener is a DTLS listener that provides accept with context.
type DTLSListener struct {
	listener         net.Listener
	config           *dtls.Config
	closed           atomic.Bool
	goPool           GoPoolFunc
	acceptedConnChan chan acceptedConn
	wg               sync.WaitGroup
	done             chan struct{}
}

func tlsPacketFilter(packet []byte) bool {
	pkts, err := recordlayer.UnpackDatagram(packet)
	if err != nil || len(pkts) < 1 {
		return false
	}
	h := &recordlayer.Header{}
	if err := h.Unmarshal(pkts[0]); err != nil {
		return false
	}
	return h.ContentType == protocol.ContentTypeHandshake
}

// NewDTLSListener creates dtls listener.
// Known networks are "udp", "udp4" (IPv4-only), "udp6" (IPv6-only).
func NewDTLSListener(network string, addr string, dtlsCfg *dtls.Config, opts ...DTLSListenerOption) (*DTLSListener, error) {
	a, err := net.ResolveUDPAddr(network, addr)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve address: %w", err)
	}
	cfg := DefaultDTLSListenerConfig
	for _, o := range opts {
		o.ApplyDTLS(&cfg)
	}

	if cfg.GoPool == nil {
		return nil, fmt.Errorf("empty go pool")
	}

	l := DTLSListener{
		goPool:           cfg.GoPool,
		config:           dtlsCfg,
		acceptedConnChan: make(chan acceptedConn, 256),
		done:             make(chan struct{}),
	}
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

	lc := udp.ListenConfig{
		AcceptFilter: tlsPacketFilter,
	}
	l.listener, err = lc.Listen(network, a)
	if err != nil {
		return nil, err
	}
	l.wg.Add(1)
	go l.run()
	return &l, nil
}

func (l *DTLSListener) send(conn net.Conn, err error) {
	select {
	case <-l.done:
	case l.acceptedConnChan <- acceptedConn{
		conn: conn,
		err:  err,
	}:
	}
}

func (l *DTLSListener) accept() error {
	c, err := l.listener.Accept()
	if err != nil {
		l.send(nil, err)
		return err
	}
	err = l.goPool(func() {
		l.send(dtls.Server(dtlsnet.PacketConnFromConn(c), c.RemoteAddr(), l.config))
	})
	if err != nil {
		_ = c.Close()
	}
	return err
}

func (l *DTLSListener) run() {
	defer l.wg.Done()
	for {
		if l.closed.Load() {
			return
		}
		err := l.accept()
		if errors.Is(err, udp.ErrClosedListener) {
			return
		}
	}
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
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-l.done:
			return nil, ErrListenerIsClosed
		case d := <-l.acceptedConnChan:
			err := d.err
			if errors.Is(err, context.DeadlineExceeded) {
				// we don't want to report error handshake deadline exceeded
				continue
			}
			if errors.Is(err, udp.ErrClosedListener) {
				return nil, ErrListenerIsClosed
			}
			if err != nil {
				return nil, err
			}
			return d.conn, nil
		}
	}
}

// Accept waits for a generic Conn.
func (l *DTLSListener) Accept() (net.Conn, error) {
	return l.AcceptWithContext(context.Background())
}

// Close closes the connection.
func (l *DTLSListener) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(l.done)
	defer l.wg.Wait()
	return l.listener.Close()
}

// Addr represents a network end point address.
func (l *DTLSListener) Addr() net.Addr {
	return l.listener.Addr()
}
