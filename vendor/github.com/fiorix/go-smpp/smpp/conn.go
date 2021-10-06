// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package smpp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/fiorix/go-smpp/smpp/pdu"
)

var (
	// ErrNotConnected is returned on attempts to use a dead connection.
	ErrNotConnected = errors.New("not connected")

	// ErrNotBound is returned on attempts to use a Transmitter,
	// Receiver or Transceiver before calling Bind.
	ErrNotBound = errors.New("not bound")

	// ErrTimeout is returned when we've reached timeout while waiting for response.
	ErrTimeout = errors.New("timeout waiting for response")
)

// Conn is an SMPP connection.
type Conn interface {
	Reader
	Writer
	Closer
}

// Reader is the interface that wraps the basic Read method.
type Reader interface {
	// Read reads PDU binary data off the wire and returns it.
	Read() (pdu.Body, error)
}

// Writer is the interface that wraps the basic Write method.
type Writer interface {
	// Write serializes the given PDU and writes to the connection.
	Write(w pdu.Body) error
}

// Closer is the interface that wraps the basic Close method.
type Closer interface {
	// Close terminates the connection.
	Close() error
}

// Dial dials to the SMPP server and returns a Conn, or error.
// TLS is only used if provided.
func Dial(addr string, TLS *tls.Config) (Conn, error) {
	if addr == "" {
		addr = "localhost:2775"
	}
	fd, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if TLS != nil {
		fd = tls.Client(fd, TLS)
	}
	c := &conn{
		rwc: fd,
		r:   bufio.NewReader(fd),
		w:   bufio.NewWriter(fd),
	}
	return c, nil
}

// conn provides the basics of a single client connection and
// implements the Conn interface.
type conn struct {
	rwc net.Conn
	r   *bufio.Reader
	w   *bufio.Writer
}

// Read implements the Conn interface.
func (c *conn) Read() (pdu.Body, error) {
	return pdu.Decode(c.r)
}

// Write implements the Conn interface.
func (c *conn) Write(w pdu.Body) error {
	var b bytes.Buffer
	err := w.SerializeTo(&b)
	if err != nil {
		return err
	}
	_, err = io.Copy(c.w, &b)
	if err != nil {
		return err
	}
	return c.w.Flush()
}

// Close implements the Conn interface.
func (c *conn) Close() error {
	return c.rwc.Close()
}

// connSwitch implements the Conn interface but allows switching
// the actual Conn object it wraps.
//
// If no Conn is available, any attempt to Read/Write/Close
// returns ErrNotConnected.
type connSwitch struct {
	mu sync.Mutex
	c  Conn
}

// Set sets the underlying Conn with the given one.
// If we hold a Conn already, it will be closed before switching over.
func (cs *connSwitch) Set(c Conn) {
	cs.mu.Lock()
	if cs.c != nil {
		cs.c.Close()
	}
	cs.c = c
	cs.mu.Unlock()
}

// Read implements the Conn interface.
func (cs *connSwitch) Read() (pdu.Body, error) {
	cs.mu.Lock()
	conn := cs.c
	cs.mu.Unlock()
	if conn == nil {
		return nil, ErrNotConnected
	}
	return conn.Read()
}

// Write implements the Conn interface.
func (cs *connSwitch) Write(w pdu.Body) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.c == nil {
		return ErrNotConnected
	}
	return cs.c.Write(w)
}

// Close implements the Conn interface.
func (cs *connSwitch) Close() error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	if cs.c == nil {
		return ErrNotConnected
	}
	err := cs.c.Close()
	cs.c = nil
	return err
}
