package net

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Conn is a generic stream-oriented network connection that provides Read/Write with context.
//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type Conn struct {
	heartBeat      time.Duration
	connection     net.Conn
	onReadTimeout  func() error
	onWriteTimeout func() error

	handshake  func() error
	readBuffer *bufio.Reader
	lock       sync.Mutex
}

var defaultConnOptions = connOptions{
	heartBeat: time.Millisecond * 200,
}

type connOptions struct {
	heartBeat      time.Duration
	onReadTimeout  func() error
	onWriteTimeout func() error
}

// A ConnOption sets options such as heartBeat, errors parameters, etc.
type ConnOption interface {
	applyConn(*connOptions)
}

// NewConn creates connection over net.Conn.
func NewConn(c net.Conn, opts ...ConnOption) *Conn {
	cfg := defaultConnOptions
	for _, o := range opts {
		o.applyConn(&cfg)
	}
	connection := Conn{
		connection:     c,
		heartBeat:      cfg.heartBeat,
		readBuffer:     bufio.NewReaderSize(c, 2048),
		onReadTimeout:  cfg.onReadTimeout,
		onWriteTimeout: cfg.onWriteTimeout,
	}
	if v, ok := c.(interface{ Handshake() error }); ok {
		connection.handshake = v.Handshake
	}

	return &connection
}

// LocalAddr returns the local network address. The Addr returned is shared by all invocations of LocalAddr, so do not modify it.
func (c *Conn) LocalAddr() net.Addr {
	return c.connection.LocalAddr()
}

// Connection returns the network connection. The Conn returned is shared by all invocations of Connection, so do not modify it.
func (c *Conn) Connection() net.Conn {
	return c.connection
}

// RemoteAddr returns the remote network address. The Addr returned is shared by all invocations of RemoteAddr, so do not modify it.
func (c *Conn) RemoteAddr() net.Addr {
	return c.connection.RemoteAddr()
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.connection.Close()
}

// WriteWithContext writes data with context.
func (c *Conn) WriteWithContext(ctx context.Context, data []byte) error {
	written := 0
	c.lock.Lock()
	defer c.lock.Unlock()
	for written < len(data) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := c.doHandshakeLocked(ctx, c.onWriteTimeout)
		if err != nil {
			return fmt.Errorf("cannot TLS handshake: %w", err)
		}
		deadline := time.Now().Add(c.heartBeat)
		err = c.connection.SetWriteDeadline(deadline)
		if err != nil {
			return fmt.Errorf("cannot set write deadline for connection: %w", err)
		}
		n, err := c.connection.Write(data[written:])

		if err != nil {
			if isTemporary(err, deadline) {
				if n > 0 {
					written += n
				}
				if c.onWriteTimeout != nil {
					err := c.onWriteTimeout()
					if err != nil {
						return fmt.Errorf("cannot write to connection: on timeout returns error: %w", err)
					}
				}
				continue
			}
			return err
		}
		written += n
	}
	return nil
}

// ReadFullWithContext reads stream with context until whole buffer is satisfied.
func (c *Conn) ReadFullWithContext(ctx context.Context, buffer []byte) error {
	offset := 0
	for offset < len(buffer) {
		n, err := c.ReadWithContext(ctx, buffer[offset:])
		if err != nil {
			return fmt.Errorf("cannot read full from connection: %w", err)
		}
		offset += n
	}
	return nil
}

// During handshake wee need to use setDeadline because https://github.com/golang/go/issues/31224
// added comment in https://github.com/golang/go/commit/c9b9cd73bb7a7828d34f4a7844f16c3fbc0674dd
func (c *Conn) doHandshakeLocked(ctx context.Context, onTimeout func() error) error {
	if c.handshake == nil {
		return nil
	}
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("handshake failed")
		default:
		}
		deadline := time.Now().Add(c.heartBeat)
		err := c.connection.SetDeadline(deadline)
		if err != nil {
			return fmt.Errorf("cannot set deadline for handshake: %w", err)
		}
		err = c.handshake()
		if err != nil {
			if isTemporary(err, deadline) {
				if onTimeout != nil {
					err := onTimeout()
					if err != nil {
						return fmt.Errorf("on timeout returns error: %w", err)
					}
				}
				continue
			}
		}
		return err
	}
}

func (c *Conn) doHandshake(ctx context.Context, onTimeout func() error) error {
	if c.handshake == nil {
		return nil
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.doHandshakeLocked(ctx, onTimeout)
}

// ReadWithContext reads stream with context.
func (c *Conn) ReadWithContext(ctx context.Context, buffer []byte) (int, error) {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return -1, ctx.Err()
			}
			return -1, fmt.Errorf("cannot read from connection")
		default:
		}
		err := c.doHandshake(ctx, c.onReadTimeout)
		if err != nil {
			return -1, fmt.Errorf("cannot TLS handshake: %w", err)
		}
		deadline := time.Now().Add(c.heartBeat)
		err = c.connection.SetReadDeadline(deadline)
		if err != nil {
			return -1, fmt.Errorf("cannot set read deadline for connection: %w", err)
		}
		n, err := c.readBuffer.Read(buffer)
		if err != nil {
			if isTemporary(err, deadline) {
				if c.onReadTimeout != nil {
					err := c.onReadTimeout()
					if err != nil {
						return -1, fmt.Errorf("cannot read from connection: on timeout returns error: %w", err)
					}
				}
				continue
			}
			return -1, err
		}
		return n, err
	}
}
