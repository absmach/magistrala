// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package nats contains NATS message publisher implementation.
package nats

import (
	"context"
	"fmt"

	"github.com/sony/gobreaker"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/ws"
	broker "github.com/nats-io/nats.go"
)

const (
	prefix          = "channel"
	maxFailedReqs   = 3
	maxFailureRatio = 0.6
)

var _ ws.Service = (*natsPubSub)(nil)

type natsPubSub struct {
	nc     *broker.Conn
	cb     *gobreaker.CircuitBreaker
	logger log.Logger
}

// New instantiates NATS message publisher.
func New(nc *broker.Conn, logger log.Logger) ws.Service {
	st := gobreaker.Settings{
		Name: "NATS",
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			fr := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= maxFailedReqs && fr >= maxFailureRatio
		},
	}
	cb := gobreaker.NewCircuitBreaker(st)
	return &natsPubSub{
		nc:     nc,
		cb:     cb,
		logger: logger,
	}
}

func (pubsub *natsPubSub) fmtSubject(chanID, subtopic string) string {
	subject := fmt.Sprintf("%s.%s", prefix, chanID)
	if subtopic != "" {
		subject = fmt.Sprintf("%s.%s", subject, subtopic)
	}
	return subject
}

func (pubsub *natsPubSub) Publish(_ context.Context, _ string, msg mainflux.Message) error {
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := pubsub.fmtSubject(msg.Channel, msg.Subtopic)
	return pubsub.nc.Publish(subject, data)
}

func (pubsub *natsPubSub) Subscribe(chanID, subtopic string, channel *ws.Channel) error {
	var sub *broker.Subscription

	sub, err := pubsub.nc.Subscribe(pubsub.fmtSubject(chanID, subtopic), func(msg *broker.Msg) {
		if msg == nil {
			pubsub.logger.Warn("Received nil message")
			return
		}

		var m mainflux.Message
		if err := proto.Unmarshal(msg.Data, &m); err != nil {
			pubsub.logger.Warn(fmt.Sprintf("Failed to deserialize received message: %s", err.Error()))
			return
		}

		pubsub.logger.Debug(fmt.Sprintf("Successfully received message from NATS from channel %s", m.GetChannel()))

		// Sends message to messages channel
		channel.Send(m)
	})

	// Check if subscription should be closed
	go func() {
		<-channel.Closed
		sub.Unsubscribe()
	}()

	return err
}
