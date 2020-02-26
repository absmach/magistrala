// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package subscriber

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/twins"
	nats "github.com/nats-io/nats.go"
)

const (
	queue = "twins"
	input = "channel.>"
)

// Subscriber is used to intercept messages and save corresponding twin states
type Subscriber struct {
	natsClient *nats.Conn
	logger     log.Logger
	svc        twins.Service
	channelID  string
}

// NewSubscriber instances Subscriber strucure and subscribes to appropriate NATS topic
func NewSubscriber(nc *nats.Conn, chID string, svc twins.Service, logger log.Logger) *Subscriber {
	s := Subscriber{
		natsClient: nc,
		logger:     logger,
		svc:        svc,
		channelID:  chID,
	}

	if _, err := s.natsClient.QueueSubscribe(input, queue, s.handleMsg); err != nil {
		logger.Error(fmt.Sprintf("Failed to subscribe to NATS: %s", err))
		os.Exit(1)
	}

	return &s
}

func (s *Subscriber) handleMsg(m *nats.Msg) {
	var msg mainflux.Message
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		s.logger.Warn(fmt.Sprintf("Unmarshalling failed: %s", err))
		return
	}

	if msg.Channel == s.channelID {
		return
	}

	if err := s.svc.SaveStates(&msg); err != nil {
		s.logger.Error(fmt.Sprintf("State save failed: %s", err))
		return
	}
}
