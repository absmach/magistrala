//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality.
package ws

import (
	"context"
	"errors"
	"sync"

	"github.com/mainflux/mainflux"
	broker "github.com/nats-io/go-nats"
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")
)

// Service specifies web socket service API.
type Service interface {
	mainflux.MessagePublisher

	// Subscribes to channel with specified id.
	Subscribe(string, string, *Channel) error
}

// Channel is used for receiving and sending messages.
type Channel struct {
	Messages chan mainflux.RawMessage
	Closed   chan bool
	closed   bool
	mutex    sync.Mutex
}

// NewChannel instantiates empty channel.
func NewChannel() *Channel {
	return &Channel{
		Messages: make(chan mainflux.RawMessage),
		Closed:   make(chan bool),
		closed:   false,
		mutex:    sync.Mutex{},
	}
}

// Send method send message over Messages channel.
func (channel *Channel) Send(msg mainflux.RawMessage) {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()

	if !channel.closed {
		channel.Messages <- msg
	}
}

// Close channel and stop message transfer.
func (channel *Channel) Close() {
	channel.mutex.Lock()
	defer channel.mutex.Unlock()

	channel.closed = true
	channel.Closed <- true
	close(channel.Messages)
	close(channel.Closed)
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	pubsub Service
}

// New instantiates the WS adapter implementation.
func New(pubsub Service) Service {
	return &adapterService{pubsub: pubsub}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg mainflux.RawMessage) error {
	if err := as.pubsub.Publish(ctx, token, msg); err != nil {
		switch err {
		case broker.ErrConnectionClosed, broker.ErrInvalidConnection:
			return ErrFailedConnection
		default:
			return ErrFailedMessagePublish
		}
	}
	return nil
}

func (as *adapterService) Subscribe(chanID, subtopic string, channel *Channel) error {
	if err := as.pubsub.Subscribe(chanID, subtopic, channel); err != nil {
		return ErrFailedSubscription
	}
	return nil
}
