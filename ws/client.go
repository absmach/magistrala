// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package ws

import (
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/gorilla/websocket"
)

// Client handles messaging and websocket connection.
type Client struct {
	conn *websocket.Conn
	id   string
}

// NewClient returns a new websocket client.
func NewClient(c *websocket.Conn) *Client {
	return &Client{
		conn: c,
		id:   "",
	}
}

// Cancel handles the websocket connection after unsubscribing.
func (c *Client) Cancel() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Handle handles the sending and receiving of messages via the broker.
func (c *Client) Handle(msg *messaging.Message) error {
	// To prevent publisher from receiving its own published message
	if msg.GetPublisher() == c.id {
		return nil
	}

	return c.conn.WriteMessage(websocket.TextMessage, msg.GetPayload())
}
