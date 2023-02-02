// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	mux "github.com/plgd-dev/go-coap/v2/mux"
)

// Client wraps CoAP client.
type Client interface {
	// In CoAP terminology, Token similar to the Session ID.
	Token() string
	Handle(m *messaging.Message) error
	Cancel() error
	Done() <-chan struct{}
}

// ErrOption indicates an error when adding an option.
var ErrOption = errors.New("unable to set option")

type client struct {
	client  mux.Client
	token   message.Token
	observe uint32
	logger  logger.Logger
}

// NewClient instantiates a new Observer.
func NewClient(c mux.Client, tkn message.Token, l logger.Logger) Client {
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
		Body:    bytes.NewReader(msg.Payload),
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
	opts, n, err = opts.SetObserve(buff, uint32(c.observe))
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
