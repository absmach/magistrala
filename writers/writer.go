// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writers

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/normalizer"
	nats "github.com/nats-io/go-nats"
)

type consumer struct {
	nc         *nats.Conn
	channels   map[string]bool
	repo       MessageRepository
	normalizer normalizer.Service
	logger     log.Logger
}

// Start method starts to consume normalized messages received from NATS.
func Start(nc *nats.Conn, repo MessageRepository, norm normalizer.Service, queue string, channels map[string]bool, logger log.Logger) error {
	c := consumer{
		nc:         nc,
		channels:   channels,
		repo:       repo,
		normalizer: norm,
		logger:     logger,
	}

	_, err := nc.QueueSubscribe(mainflux.InputChannels, queue, c.consume)
	return err
}

func (c *consumer) consume(m *nats.Msg) {
	var msg mainflux.RawMessage
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	norm, err := c.normalizer.Normalize(msg)
	if err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to normalize received message: %s", err))
		return
	}
	var msgs []mainflux.Message
	for _, v := range norm {
		if c.channelExists(v.GetChannel()) {
			msgs = append(msgs, v)
		}
	}

	if err := c.repo.Save(msgs...); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}

func (c *consumer) channelExists(channel string) bool {
	if _, ok := c.channels["*"]; ok {
		return true
	}

	_, found := c.channels[channel]
	return found
}
