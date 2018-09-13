//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package writers

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	nats "github.com/nats-io/go-nats"
)

type consumer struct {
	nc     *nats.Conn
	logger log.Logger
	repo   MessageRepository
}

// Start method starts to consume normalized messages received from NATS.
func Start(nc *nats.Conn, repo MessageRepository, queue string, logger log.Logger) error {
	c := consumer{
		nc:     nc,
		logger: logger,
		repo:   repo,
	}

	_, err := nc.QueueSubscribe(mainflux.OutputSenML, queue, c.consume)
	return err
}

func (c *consumer) consume(m *nats.Msg) {
	msg := &mainflux.Message{}

	if err := proto.Unmarshal(m.Data, msg); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	if err := c.repo.Save(*msg); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}
