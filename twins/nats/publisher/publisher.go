// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package publisher

import (
	"fmt"

	log "github.com/mainflux/mainflux/logger"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	"github.com/nats-io/nats.go"
)

const (
	prefix    = "channel"
	publisher = "twins"
)

// Publisher is used to publish twins related notifications
type Publisher struct {
	natsClient *nats.Conn
	channelID  string
	logger     log.Logger
}

// NewPublisher instances Pubsub strucure
func NewPublisher(nc *nats.Conn, chID string, logger log.Logger) *Publisher {
	return &Publisher{
		natsClient: nc,
		channelID:  chID,
		logger:     logger,
	}
}

// Publish sends twins CRUD and state saving related operations
func (p *Publisher) Publish(twinID *string, err *error, succOp, failOp string, payload *[]byte) {
	if p.channelID == "" {
		return
	}

	op := succOp
	if *err != nil {
		op = failOp
		esb := []byte((*err).Error())
		payload = &esb
	}

	pl := *payload
	if pl == nil {
		pl = []byte(fmt.Sprintf("{\"deleted\":\"%s\"}", *twinID))
	}
	subject := fmt.Sprintf("%s.%s.%s", prefix, p.channelID, op)
	mc := mainflux.Message{
		Channel:   p.channelID,
		Subtopic:  op,
		Payload:   pl,
		Publisher: publisher,
	}
	b, _ := proto.Marshal(&mc)

	if err := p.natsClient.Publish(subject, []byte(b)); err != nil {
		p.logger.Warn(fmt.Sprintf("Failed to publish notification on NATS: %s", err))
	}
}
