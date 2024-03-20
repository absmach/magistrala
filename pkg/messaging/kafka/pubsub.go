// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/go-zookeeper/zk"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	chansPrefix                  = "channels"
	SubjectAllChannels           = "channels.*"
	offset                       = kafka.LastOffset
	defaultScanningInterval      = 10 * time.Millisecond
	topicsRoot                   = "/brokers/topics"
	zkTimeout                    = time.Second
	readerWaitTime               = time.Second
	readerHeartbeatInterval      = time.Second
	readerPartitionWatchInterval = time.Second
	readerSessionTimeout         = time.Second * 10
)

var (
	ErrAlreadySubscribed = errors.New("already subscribed to topic")
	ErrNotSubscribed     = errors.New("not subscribed")
	ErrEmptyTopic        = errors.New("empty topic")
	ErrEmptyID           = errors.New("empty id")
)

var _ messaging.PubSub = (*pubsub)(nil)

type subscription struct {
	*kafka.Reader
	cancel func() error
}
type pubsub struct {
	publisher
	zkConn        *zk.Conn
	logger        *slog.Logger
	mu            sync.Mutex
	subscriptions map[string]map[string]subscription
}

// NewPubSub returns Kafka message publisher/subscriber.
func NewPubSub(url, _ string, logger *slog.Logger) (messaging.PubSub, error) {
	conn, err := kafka.Dial("tcp", url)
	if err != nil {
		return &pubsub{}, err
	}
	host, _, err := net.SplitHostPort(url)
	if err != nil {
		return &pubsub{}, fmt.Errorf("failed to split host and port: %w", err)
	}

	zkConn, _, err := zk.Connect([]string{host}, zkTimeout, zk.WithLogInfo(false))
	if err != nil {
		return &pubsub{}, fmt.Errorf("failed to connect to zookeeper: %w", err)
	}

	return &pubsub{
		publisher: publisher{
			url:    url,
			conn:   conn,
			topics: make(map[string]*kafka.Writer),
		},
		zkConn:        zkConn,
		subscriptions: make(map[string]map[string]subscription),
		logger:        logger,
	}, nil
}

func (ps *pubsub) Subscribe(ctx context.Context, cfg messaging.SubscriberConfig) error {
	if cfg.ID == "" {
		return ErrEmptyID
	}
	if cfg.Topic == "" {
		return ErrEmptyTopic
	}
	cfg.Topic = formatTopic(cfg.Topic)

	if cfg.Topic == SubjectAllChannels {
		go ps.subscribeToAllChannels(ctx, cfg.ID, cfg.Handler)
		return nil
	}

	s, err := ps.checkSubscribed(cfg.Topic, cfg.ID)
	if err != nil {
		return err
	}
	ps.configReader(ctx, cfg.ID, cfg.Topic, s, cfg.Handler)

	return nil
}

func (ps *pubsub) Unsubscribe(ctx context.Context, id, topic string) error {
	if id == "" {
		return ErrEmptyID
	}
	if topic == "" {
		return ErrEmptyTopic
	}
	topic = formatTopic(topic)

	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Check topic
	subs, ok := ps.subscriptions[topic]
	if !ok {
		return ErrNotSubscribed
	}
	// Check topic ID
	s, ok := subs[id]
	if !ok {
		return ErrNotSubscribed
	}
	if s.cancel != nil {
		if err := s.cancel(); err != nil {
			return err
		}
	}
	delete(subs, id)
	if len(subs) == 0 {
		delete(ps.subscriptions, topic)
	}
	return nil
}

func (ps *pubsub) Close() error {
	defer ps.zkConn.Close()

	for _, subs := range ps.subscriptions {
		for _, s := range subs {
			if s.cancel != nil {
				if err := s.cancel(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ps *pubsub) handle(message kafka.Message, h messaging.MessageHandler) {
	var msg messaging.Message
	if err := proto.Unmarshal(message.Value, &msg); err != nil {
		ps.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
	}
	if err := h.Handle(&msg); err != nil {
		ps.logger.Warn(fmt.Sprintf("Failed to handle Mainflux message: %s", err))
	}
}

// checkSubscribed checks if topic and topic ID are already subscribed.
func (ps *pubsub) checkSubscribed(topic, id string) (map[string]subscription, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// Check topic
	s, ok := ps.subscriptions[topic]
	switch ok {
	case true:
		// Check topic ID
		if _, ok := s[id]; ok {
			return map[string]subscription{}, ErrAlreadySubscribed
		}
	default:
		s = make(map[string]subscription)
		ps.subscriptions[topic] = s
	}
	return s, nil
}

// configReader configures reader for given topic and starts consuming messages.
func (ps *pubsub) configReader(ctx context.Context, id, topic string, s map[string]subscription, handler messaging.MessageHandler) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:                []string{ps.url},
		GroupID:                id,
		Topic:                  topic,
		StartOffset:            offset,
		MaxWait:                readerWaitTime,
		HeartbeatInterval:      readerHeartbeatInterval,
		PartitionWatchInterval: readerPartitionWatchInterval,
		SessionTimeout:         readerSessionTimeout,
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				message, err := reader.ReadMessage(ctx)
				if err == io.EOF {
					// This is because the reader has been closed.
					return
				}
				if err != nil {
					ps.logger.Warn(fmt.Sprintf("Failed to read message: %s", err))
					continue
				}
				ps.handle(message, handler)
			}
		}
	}()
	s[id] = subscription{
		Reader: reader,
		cancel: handler.Cancel,
	}
}

// subscribeToAllChannels subscribes to all channels by prediocially scanning for all topics and consuming them.
func (ps *pubsub) subscribeToAllChannels(ctx context.Context, id string, handler messaging.MessageHandler) {
	ticker := time.NewTicker(defaultScanningInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			topics, err := ps.listAllTopics()
			if err != nil {
				ps.logger.Warn(fmt.Sprintf("Failed to list topics: %s", err))
				continue
			}

			for _, t := range topics {
				s, err := ps.checkSubscribed(t, id)
				if err == nil {
					ps.configReader(ctx, id, t, s, handler)
				}
			}
		}
	}
}

// listAllTopics lists all topics in zookeeper.
func (ps *pubsub) listAllTopics() ([]string, error) {
	var topics []string

	children, _, err := ps.zkConn.Children(topicsRoot)
	if err != nil {
		return topics, err
	}

	for _, topic := range children {
		if strings.HasPrefix(topic, chansPrefix) {
			topics = append(topics, topic)
		}
	}

	return topics, nil
}
