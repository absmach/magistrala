// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/messaging"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
)

const username = "magistrala-mqtt"

var (
	// ErrConnect indicates that connection to MQTT broker failed.
	ErrConnect = errors.New("failed to connect to MQTT broker")

	// ErrSubscribeTimeout indicates that the subscription failed due to timeout.
	ErrSubscribeTimeout = errors.New("failed to subscribe due to timeout reached")

	// ErrUnsubscribeTimeout indicates that unsubscribe failed due to timeout.
	ErrUnsubscribeTimeout = errors.New("failed to unsubscribe due to timeout reached")

	// ErrUnsubscribeDeleteTopic indicates that unsubscribe failed because the topic was deleted.
	ErrUnsubscribeDeleteTopic = errors.New("failed to unsubscribe due to deletion of topic")

	// ErrNotSubscribed indicates that the topic is not subscribed to.
	ErrNotSubscribed = errors.New("not subscribed")

	// ErrEmptyTopic indicates the absence of topic.
	ErrEmptyTopic = errors.New("empty topic")

	// ErrEmptyID indicates the absence of ID.
	ErrEmptyID = errors.New("empty ID")
)

var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	client mqtt.Client
	topics []string
	cancel func() error
}

type pubsub struct {
	publisher
	logger        mglog.Logger
	mu            sync.RWMutex
	address       string
	timeout       time.Duration
	subscriptions map[string]subscription
}

// NewPubSub returns MQTT message publisher/subscriber.
func NewPubSub(url string, qos uint8, timeout time.Duration, logger mglog.Logger) (messaging.PubSub, error) {
	client, err := newClient(url, "mqtt-publisher", timeout)
	if err != nil {
		return nil, err
	}
	ret := &pubsub{
		publisher: publisher{
			client:  client,
			timeout: timeout,
			qos:     qos,
		},
		address:       url,
		timeout:       timeout,
		logger:        logger,
		subscriptions: make(map[string]subscription),
	}
	return ret, nil
}

func (ps *pubsub) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	s, ok := ps.subscriptions[cfg.ID]
	// If the client exists, check if it's subscribed to the topic and unsubscribe if needed.
	switch ok {
	case true:
		if ok := s.contains(cfg.Topic); ok {
			if err := s.unsubscribe(cfg.Topic, ps.timeout); err != nil {
				return err
			}
		}
	default:
		client, err := newClient(ps.address, cfg.ID, ps.timeout)
		if err != nil {
			return err
		}
		s = subscription{
			client: client,
			topics: []string{},
			cancel: cfg.Handler.Cancel,
		}
	}
	s.topics = append(s.topics, cfg.Topic)
	ps.subscriptions[cfg.ID] = s

	token := s.client.Subscribe(cfg.Topic, byte(ps.qos), ps.mqttHandler(cfg.Handler))
	if token.Error() != nil {
		return token.Error()
	}
	if ok := token.WaitTimeout(ps.timeout); !ok {
		return ErrSubscribeTimeout
	}

	return nil
}

func (ps *pubsub) Unsubscribe(ctx context.Context, id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()

	s, ok := ps.subscriptions[id]
	if !ok || !s.contains(topic) {
		return ErrNotSubscribed
	}

	if err := s.unsubscribe(topic, ps.timeout); err != nil {
		return err
	}
	ps.subscriptions[id] = s

	if len(s.topics) == 0 {
		delete(ps.subscriptions, id)
	}
	return nil
}

func (s *subscription) unsubscribe(topic string, timeout time.Duration) error {
	if s.cancel != nil {
		if err := s.cancel(); err != nil {
			return err
		}
	}

	token := s.client.Unsubscribe(topic)
	if token.Error() != nil {
		return token.Error()
	}

	if ok := token.WaitTimeout(timeout); !ok {
		return ErrUnsubscribeTimeout
	}
	if ok := s.delete(topic); !ok {
		return ErrUnsubscribeDeleteTopic
	}
	return token.Error()
}

func newClient(address, id string, timeout time.Duration) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().
		SetUsername(username).
		AddBroker(address).
		SetClientID(id)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Error() != nil {
		return nil, token.Error()
	}

	if ok := token.WaitTimeout(timeout); !ok {
		return nil, ErrConnect
	}

	return client, nil
}

func (ps *pubsub) mqttHandler(h messaging.MessageHandler) mqtt.MessageHandler {
	return func(_ mqtt.Client, m mqtt.Message) {
		var msg messaging.Message
		if err := proto.Unmarshal(m.Payload(), &msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
			return
		}

		if err := h.Handle(&msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle Magistrala message: %s", err))
		}
	}
}

// Contains checks if a topic is present.
func (s subscription) contains(topic string) bool {
	return s.indexOf(topic) != -1
}

// Finds the index of an item in the topics.
func (s subscription) indexOf(element string) int {
	for k, v := range s.topics {
		if element == v {
			return k
		}
	}
	return -1
}

// Deletes a topic from the slice.
func (s *subscription) delete(topic string) bool {
	index := s.indexOf(topic)
	if index == -1 {
		return false
	}
	topics := make([]string, len(s.topics)-1)
	copy(topics[:index], s.topics[:index])
	copy(topics[index:], s.topics[index+1:])
	s.topics = topics
	return true
}
