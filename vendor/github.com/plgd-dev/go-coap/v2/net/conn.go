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
	heartBeat  time.Duration
	connection net.Conn
	readBuffer *bufio.Reader
	lock       sync.Mutex
}

var defaultConnOptions = connOptions{
	heartBeat: time.Millisecond * 200,
}

type connOptions struct {
	heartBeat time.Duration
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
		connection: c,
		heartBeat:  cfg.heartBeat,
		readBuffer: bufio.NewReaderSize(c, 2048),
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

// WriteContext writes data with context.
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
		err := c.connection.SetWriteDeadline(time.Now().Add(c.heartBeat))
		if err != nil {
			return fmt.Errorf("cannot set write deadline for tcp connection: %w", err)
		}
		n, err := c.connection.Write(data[written:])

		if err != nil {
			if isTemporary(err) {
				continue
			}
			return fmt.Errorf("cannot write to tcp connection")
		}
		written += n
	}
	return nil
}

// ReadFullContext reads stream with context until whole buffer is satisfied.
func (c *Conn) ReadFullWithContext(ctx context.Context, buffer []byte) error {
	offset := 0
	for offset < len(buffer) {
		n, err := c.ReadWithContext(ctx, buffer[offset:])
		if err != nil {
			return fmt.Errorf("cannot read full from tcp connection: %w", err)
		}
		offset += n
	}
	return nil
}

// ReadContext reads stream with context.
func (c *Conn) ReadWithContext(ctx context.Context, buffer []byte) (int, error) {
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				return -1, fmt.Errorf("cannot read from tcp connection: %v", ctx.Err())
			}
			return -1, fmt.Errorf("cannot read from tcp connection")
		default:
		}

		err := c.connection.SetReadDeadline(time.Now().Add(c.heartBeat))
		if err != nil {
			return -1, fmt.Errorf("cannot set read deadline for tcp connection: %w", err)
		}
		n, err := c.readBuffer.Read(buffer)
		if err != nil {
			if isTemporary(err) {
				continue
			}
			return -1, fmt.Errorf("cannot read from tcp connection: %w", err)
		}
		return n, err
	}
}
