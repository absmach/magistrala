// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rabbitmq

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gogo/protobuf/proto"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	chansPrefix = "channels"
	// SubjectAllChannels represents subject to subscribe for all the channels.
	SubjectAllChannels = "channels.#"
	exchangeName       = "mainflux-exchange"
)

var (
	// ErrNotSubscribed indicates that the topic is not subscribed to.
	ErrNotSubscribed = errors.New("not subscribed")

	// ErrEmptyTopic indicates the absence of topic.
	ErrEmptyTopic = errors.New("empty topic")

	// ErrEmptyID indicates the absence of ID.
	ErrEmptyID = errors.New("empty ID")
)
var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	cancel func() error
}
type pubsub struct {
	publisher
	logger        log.Logger
	subscriptions map[string]map[string]subscription
	mu            sync.Mutex
}

// NewPubSub returns RabbitMQ message publisher/subscriber.
func NewPubSub(url, queue string, logger log.Logger) (messaging.PubSub, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchangeName, amqp.ExchangeTopic, true, false, false, false, nil); err != nil {
		return nil, err
	}
	ret := &pubsub{
		publisher: publisher{
			conn: conn,
			ch:   ch,
		},
		logger:        logger,
		subscriptions: make(map[string]map[string]subscription),
	}
	return ret, nil
}

func (ps *pubsub) Subscribe(id, topic string, handler messaging.MessageHandler) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}
	ps.mu.Lock()

	topic = formatTopic(topic)
	// Check topic
	s, ok := ps.subscriptions[topic]
	if ok {
		// Check client ID
		if _, ok := s[id]; ok {
			// Unlocking, so that Unsubscribe() can access ps.subscriptions
			ps.mu.Unlock()
			if err := ps.Unsubscribe(id, topic); err != nil {
				return err
			}

			ps.mu.Lock()
			// value of s can be changed while ps.mu is unlocked
			s = ps.subscriptions[topic]
		}
	}
	defer ps.mu.Unlock()
	if s == nil {
		s = make(map[string]subscription)
		ps.subscriptions[topic] = s
	}

	if _, err := ps.ch.QueueDeclare(topic, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ps.ch.QueueBind(topic, topic, exchangeName, false, nil); err != nil {
		return err
	}
	clientID := fmt.Sprintf("%s-%s", topic, id)
	msgs, err := ps.ch.Consume(topic, clientID, true, false, false, false, nil)
	if err != nil {
		return err
	}
	go ps.handle(msgs, handler)
	s[id] = subscription{
		cancel: func() error {
			if err := ps.ch.Cancel(clientID, false); err != nil {
				return err
			}
			return handler.Cancel()
		},
	}

	return nil
}

func (ps *pubsub) Unsubscribe(id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	topic = formatTopic(topic)
	// Check topic
	s, ok := ps.subscriptions[topic]
	if !ok {
		return ErrNotSubscribed
	}
	// Check topic ID
	current, ok := s[id]
	if !ok {
		return ErrNotSubscribed
	}
	if current.cancel != nil {
		if err := current.cancel(); err != nil {
			return err
		}
	}
	if err := ps.ch.QueueUnbind(topic, topic, exchangeName, nil); err != nil {
		return err
	}

	delete(s, id)
	if len(s) == 0 {
		delete(ps.subscriptions, topic)
	}
	return nil
}

func (ps *pubsub) handle(deliveries <-chan amqp.Delivery, h messaging.MessageHandler) {
	for d := range deliveries {
		var msg messaging.Message
		if err := proto.Unmarshal(d.Body, &msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
			return
		}
		if err := h.Handle(msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle Mainflux message: %s", err))
			return
		}
	}
}
