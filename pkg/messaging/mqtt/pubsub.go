// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mqtt

import (
	"errors"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gogo/protobuf/proto"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
)

const (
	username = "mainflux-mqtt"
	qos      = 2
)

var (
	errConnect                = errors.New("failed to connect to MQTT broker")
	errSubscribeTimeout       = errors.New("failed to subscribe due to timeout reached")
	errUnsubscribeTimeout     = errors.New("failed to unsubscribe due to timeout reached")
	errUnsubscribeDeleteTopic = errors.New("failed to unsubscribe due to deletion of topic")
	errNotSubscribed          = errors.New("not subscribed")
	errEmptyTopic             = errors.New("empty topic")
	errEmptyID                = errors.New("empty ID")
)

var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	client mqtt.Client
	topics []string
}

type pubsub struct {
	publisher
	logger        log.Logger
	mu            *sync.RWMutex
	address       string
	timeout       time.Duration
	subscriptions map[string]subscription
}

// NewPubSub returns MQTT message publisher/subscriber.
func NewPubSub(url, queue string, timeout time.Duration, logger log.Logger) (messaging.PubSub, error) {
	client, err := newClient(url, "mqtt-publisher", timeout)
	if err != nil {
		return nil, err
	}
	ret := pubsub{
		publisher: publisher{
			client:  client,
			timeout: timeout,
		},
		address:       url,
		timeout:       timeout,
		logger:        logger,
		subscriptions: make(map[string]subscription),
	}
	return ret, nil
}

func (ps pubsub) Subscribe(id, topic string, handler messaging.MessageHandler) error {
	if id == "" {
		return errEmptyID
	}
	if topic == "" {
		return errEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Check client ID
	s, ok := ps.subscriptions[id]
	switch ok {
	case true:
		// Check topic
		if ok = s.contains(topic); ok {
			// Unlocking, so that Unsubscribe() can access ps.subscriptions
			ps.mu.Unlock()
			err := ps.Unsubscribe(id, topic)
			ps.mu.Lock() // Lock so that deferred unlock handle it
			if err != nil {
				return err
			}
			if len(ps.subscriptions) == 0 {
				client, err := newClient(ps.address, id, ps.timeout)
				if err != nil {
					return err
				}
				s = subscription{
					client: client,
					topics: []string{topic},
				}
			}
		}
		s.topics = append(s.topics, topic)
	default:
		client, err := newClient(ps.address, id, ps.timeout)
		if err != nil {
			return err
		}
		s = subscription{
			client: client,
			topics: []string{topic},
		}
	}

	token := s.client.Subscribe(topic, qos, ps.mqttHandler(handler))
	if token.Error() != nil {
		return token.Error()
	}
	if ok := token.WaitTimeout(ps.timeout); !ok {
		return errSubscribeTimeout
	}
	return token.Error()
}

func (ps pubsub) Unsubscribe(id, topic string) error {
	if id == "" {
		return errEmptyID
	}
	if topic == "" {
		return errEmptyTopic
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Check client ID
	s, ok := ps.subscriptions[id]
	switch ok {
	case true:
		// Check topic
		if ok := s.contains(topic); !ok {
			return errNotSubscribed
		}
	default:
		return errNotSubscribed
	}
	token := s.client.Unsubscribe(topic)
	if token.Error() != nil {
		return token.Error()
	}

	ok = token.WaitTimeout(ps.timeout)
	if !ok {
		return errUnsubscribeTimeout
	}
	if ok := s.delete(topic); !ok {
		return errUnsubscribeDeleteTopic
	}
	if len(s.topics) == 0 {
		delete(ps.subscriptions, id)
	}
	return token.Error()
}

func (ps pubsub) mqttHandler(h messaging.MessageHandler) mqtt.MessageHandler {
	return func(c mqtt.Client, m mqtt.Message) {
		var msg messaging.Message
		if err := proto.Unmarshal(m.Payload(), &msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
			return
		}
		if err := h.Handle(msg); err != nil {
			ps.logger.Warn(fmt.Sprintf("Failed to handle Mainflux message: %s", err))
		}
	}
}

func newClient(address, id string, timeout time.Duration) (mqtt.Client, error) {
	opts := mqtt.NewClientOptions().SetUsername(username).AddBroker(address).SetClientID(id)
	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Error() != nil {
		return nil, token.Error()
	}

	ok := token.WaitTimeout(timeout)
	if !ok {
		return nil, errConnect
	}

	if token.Error() != nil {
		return nil, token.Error()
	}

	return client, nil
}

// contains checks if a topic is present
func (sub subscription) contains(topic string) bool {
	return sub.indexOf(topic) != -1
}

// Finds the index of an item in the topics
func (sub subscription) indexOf(element string) int {
	for k, v := range sub.topics {
		if element == v {
			return k
		}
	}
	return -1
}

// Deletes a topic from the slice
func (sub subscription) delete(topic string) bool {
	index := sub.indexOf(topic)
	if index == -1 {
		return false
	}
	topics := make([]string, len(sub.topics)-1)
	copy(topics[:index], sub.topics[:index])
	copy(topics[index:], sub.topics[index+1:])
	sub.topics = topics
	return true
}
