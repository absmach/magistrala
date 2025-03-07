// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"bytes"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	mux "github.com/plgd-dev/go-coap/v3/mux"
)

// Client wraps CoAP client.
type Client interface {
	// In CoAP terminology, Token similar to the Session ID.
	Token() string

	// Handle handles incoming messages.
	Handle(m *messaging.Message) error

	// Cancel cancels the client.
	Cancel() error

	// Done returns a channel that's closed when the client is done.
	Done() <-chan struct{}
}

// ErrOption indicates an error when adding an option.
var ErrOption = errors.New("unable to set option")

type client struct {
	conn    mux.Conn
	token   message.Token
	observe uint32
	logger  *slog.Logger
}

// NewClient instantiates a new Observer.
func NewClient(conn mux.Conn, tkn message.Token, l *slog.Logger) Client {
	return &client{
		conn:    conn,
		token:   tkn,
		logger:  l,
		observe: 0,
	}
}

func (c *client) Done() <-chan struct{} {
	return c.conn.Done()
}

func (c *client) Cancel() error {
	pm := c.conn.AcquireMessage(c.conn.Context())
	pm.SetCode(codes.Content)
	pm.SetToken(c.token)
	if err := c.conn.WriteMessage(pm); err != nil {
		c.logger.Error(fmt.Sprintf("Error sending message: %s.", err))
	}
	c.conn.ReleaseMessage(pm)
	return c.conn.Close()
}

func (c *client) Token() string {
	return c.token.String()
}

func (c *client) Handle(msg *messaging.Message) error {
	pm := c.conn.AcquireMessage(c.conn.Context())
	defer c.conn.ReleaseMessage(pm)
	pm.SetCode(codes.Content)
	pm.SetToken(c.token)
	pm.SetBody(bytes.NewReader(msg.GetPayload()))

	atomic.AddUint32(&c.observe, 1)
	var opts message.Options
	var buff []byte
	opts, n, err := opts.SetContentFormat(buff, message.TextPlain)
	if err == message.ErrTooSmall {
		buff = append(buff, make([]byte, n)...)
		_, _, err = opts.SetContentFormat(buff, message.TextPlain)
	}
	if err != nil {
		c.logger.Error(fmt.Sprintf("Can't set content format: %s.", err))
		return errors.Wrap(ErrOption, err)
	}
	opts, n, err = opts.SetObserve(buff, c.observe)
	if err == message.ErrTooSmall {
		buff = append(buff, make([]byte, n)...)
		opts, _, err = opts.SetObserve(buff, uint32(c.observe))
	}
	if err != nil {
		return fmt.Errorf("cannot set options to response: %w", err)
	}

	for _, option := range opts {
		pm.SetOptionBytes(option.ID, option.Value)
	}
	return c.conn.WriteMessage(pm)
}
