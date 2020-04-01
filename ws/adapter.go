// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package ws contains the domain concept definitions needed to support
// Mainflux ws adapter service functionality.
package ws

import (
	"context"

	"errors"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/broker"
	"github.com/mainflux/mainflux/logger"
	"github.com/nats-io/nats.go"
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
	// Publish Messssage
	Publish(context.Context, string, broker.Message) error

	// Subscribes to channel with specified id.
	Subscribe(string, string, *Channel) error
}

// Channel is used for receiving and sending messages.
type Channel struct {
	Messages chan broker.Message
	Closed   chan bool
	closed   bool
	mutex    sync.Mutex
}

// NewChannel instantiates empty channel.
func NewChannel() *Channel {
	return &Channel{
		Messages: make(chan broker.Message),
		Closed:   make(chan bool),
		closed:   false,
		mutex:    sync.Mutex{},
	}
}

// Send method send message over Messages channel.
func (channel *Channel) Send(msg broker.Message) {
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
	broker broker.Nats
	log    logger.Logger
}

// New instantiates the WS adapter implementation.
func New(broker broker.Nats, log logger.Logger) Service {
	return &adapterService{
		broker: broker,
		log:    log,
	}
}

func (as *adapterService) Publish(ctx context.Context, token string, msg broker.Message) error {
	if err := as.broker.Publish(ctx, token, msg); err != nil {
		switch err {
		case nats.ErrConnectionClosed, nats.ErrInvalidConnection:
			return ErrFailedConnection
		default:
			return ErrFailedMessagePublish
		}
	}
	return nil
}

func (as *adapterService) Subscribe(chanID, subtopic string, channel *Channel) error {
	subject := chanID
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", chanID, subtopic)
	}

	sub, err := as.broker.Subscribe(subject, func(msg *nats.Msg) {
		if msg == nil {
			as.log.Warn("Received nil message")
			return
		}

		m := broker.Message{}
		if err := proto.Unmarshal(msg.Data, &m); err != nil {
			as.log.Warn(fmt.Sprintf("Failed to deserialize received message: %s", err.Error()))
			return
		}

		as.log.Debug(fmt.Sprintf("Successfully received message from NATS from channel %s", m.GetChannel()))

		// Sends message to messages channel
		channel.Send(m)
	})

	// Check if subscription should be closed
	go func() {
		<-channel.Closed
		if err := sub.Unsubscribe(); err != nil {
			as.log.Error(fmt.Sprintf("Failed to unsubscribe from %s.%s", chanID, subtopic))
		}
	}()

	return err
}
