// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writers

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	"github.com/mainflux/mainflux/transformers/senml"
	nats "github.com/nats-io/nats.go"
)

type consumer struct {
	nc          *nats.Conn
	channels    map[string]bool
	repo        MessageRepository
	transformer transformers.Transformer
	logger      log.Logger
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(nc *nats.Conn, repo MessageRepository, transformer transformers.Transformer, queue string, channels map[string]bool, logger log.Logger) error {
	c := consumer{
		nc:          nc,
		channels:    channels,
		repo:        repo,
		transformer: transformer,
		logger:      logger,
	}

	_, err := nc.QueueSubscribe(mainflux.InputChannels, queue, c.consume)
	return err
}

func (c *consumer) consume(m *nats.Msg) {
	var msg mainflux.Message
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	t, err := c.transformer.Transform(msg)
	if err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to tranform received message: %s", err))
		return
	}
	norm, ok := t.([]senml.Message)
	if !ok {
		c.logger.Warn("Invalid message format from the Transformer output.")
		return
	}
	var msgs []senml.Message
	for _, v := range norm {
		if c.channelExists(v.Channel) {
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
