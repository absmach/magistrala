// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	mux "github.com/plgd-dev/go-coap/v2/mux"
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
	client  mux.Client
	token   message.Token
	observe uint32
	logger  *slog.Logger
}

// NewClient instantiates a new Observer.
func NewClient(c mux.Client, tkn message.Token, l *slog.Logger) Client {
	return &client{
		client:  c,
		token:   tkn,
		logger:  l,
		observe: 0,
	}
}

func (c *client) Done() <-chan struct{} {
	return c.client.Done()
}

func (c *client) Cancel() error {
	m := message.Message{
		Code:    codes.Content,
		Token:   c.token,
		Context: context.Background(),
		Options: make(message.Options, 0, 16),
	}
	if err := c.client.WriteMessage(&m); err != nil {
		c.logger.Error(fmt.Sprintf("Error sending message: %s.", err))
	}
	return c.client.Close()
}

func (c *client) Token() string {
	return c.token.String()
}

func (c *client) Handle(msg *messaging.Message) error {
	m := message.Message{
		Code:    codes.Content,
		Token:   c.token,
		Context: c.client.Context(),
		Body:    bytes.NewReader(msg.GetPayload()),
	}

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
	opts = append(opts, message.Option{ID: message.Observe, Value: []byte{byte(c.observe)}})
	opts, n, err = opts.SetObserve(buff, c.observe)
	if err == message.ErrTooSmall {
		buff = append(buff, make([]byte, n)...)
		opts, _, err = opts.SetObserve(buff, uint32(c.observe))
	}
	if err != nil {
		return fmt.Errorf("cannot set options to response: %w", err)
	}

	m.Options = opts
	return c.client.WriteMessage(&m)
}
