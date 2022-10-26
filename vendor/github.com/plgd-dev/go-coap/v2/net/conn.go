package net

import (
	"context"
	"fmt"
	"net"
	"sync"

	"go.uber.org/atomic"
)

// Conn is a generic stream-oriented network connection that provides Read/Write with context.
//
// Multiple goroutines may invoke methods on a Conn simultaneously.
type Conn struct {
	connection       net.Conn
	closed           atomic.Bool
	handshakeContext func(ctx context.Context) error
	lock             sync.Mutex
}

// NewConn creates connection over net.Conn.
func NewConn(c net.Conn) *Conn {
	connection := Conn{
		connection: c,
	}

	if v, ok := c.(interface {
		HandshakeContext(ctx context.Context) error
	}); ok {
		connection.handshakeContext = v.HandshakeContext
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
	if !c.closed.CAS(false, true) {
		return nil
	}
	return c.connection.Close()
}

func (c *Conn) handshake(ctx context.Context) error {
	if c.handshakeContext != nil {
		err := c.handshakeContext(ctx)
		if err == nil {
			return nil
		}
		errC := c.Close()
		if errC == nil {
			return err
		}
		return fmt.Errorf("%v", []error{err, errC})
	}
	return nil
}

// WriteWithContext writes data with context.
func (c *Conn) WriteWithContext(ctx context.Context, data []byte) error {
	if err := c.handshake(ctx); err != nil {
		return err
	}
	written := 0
	c.lock.Lock()
	defer c.lock.Unlock()
	for written < len(data) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if c.closed.Load() {
			return ErrConnectionIsClosed
		}
		n, err := c.connection.Write(data[written:])
		if err != nil {
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

// ReadWithContext reads stream with context.
func (c *Conn) ReadWithContext(ctx context.Context, buffer []byte) (int, error) {
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	default:
	}
	if c.closed.Load() {
		return -1, ErrConnectionIsClosed
	}
	if err := c.handshake(ctx); err != nil {
		return -1, err
	}
	return c.connection.Read(buffer)
}
