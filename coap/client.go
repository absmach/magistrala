// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package coap

import (
	"bytes"
	"fmt"

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
	SendMessage(m messaging.Message) error
	Cancel() error
	Done() <-chan struct{}
}

type observers map[string]Observer

// ErrOption indicates an error when adding an option.
var ErrOption = errors.New("unable to set option")

type client struct {
	client mux.Client
	token  message.Token
	logger logger.Logger
}

// NewClient instantiates a new Observer.
func NewClient(mc mux.Client, token message.Token, l logger.Logger) Client {
	return &client{
		client: mc,
		token:  token,
		logger: l,
	}
}

func (c *client) Done() <-chan struct{} {
	return c.client.Context().Done()
}

func (c *client) Cancel() error {
	return c.client.Close()
}

func (c *client) Token() string {
	return c.token.String()
}

func (c *client) SendMessage(msg messaging.Message) error {
	m := message.Message{
		Code:    codes.Content,
		Token:   c.token,
		Context: c.client.Context(),
		Body:    bytes.NewReader(msg.Payload),
	}
	var opts message.Options
	var buff []byte

	opts, n, err := opts.SetContentFormat(buff, message.TextPlain)
	if err == message.ErrTooSmall {
		buff = append(buff, make([]byte, n)...)
		opts, n, err = opts.SetContentFormat(buff, message.TextPlain)
	}
	if err != nil {
		c.logger.Error(fmt.Sprintf("Can't set content format: %s.", err))
		return errors.Wrap(ErrOption, err)
	}
	m.Options = opts
	if err := c.client.WriteMessage(&m); err != nil {
		c.logger.Error(fmt.Sprintf("Error sending message: %s.", err))
		return err
	}
	return nil
}
