//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package nats

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/normalizer"
	"github.com/nats-io/go-nats"
)

const (
	queue         = "normalizers"
	input         = "channel.>"
	outputUnknown = "out.unknown"
	senML         = "application/senml+json"
)

type pubsub struct {
	nc     *nats.Conn
	svc    normalizer.Service
	logger log.Logger
}

// Subscribe to appropriate NATS topic and normalizes received messages.
func Subscribe(svc normalizer.Service, nc *nats.Conn, logger log.Logger) {
	ps := pubsub{
		nc:     nc,
		svc:    svc,
		logger: logger,
	}
	ps.nc.QueueSubscribe(input, queue, ps.handleMsg)
}

func (ps pubsub) handleMsg(m *nats.Msg) {
	var msg mainflux.RawMessage
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		ps.logger.Warn(fmt.Sprintf("Unmarshalling failed: %s", err))
		return
	}

	if err := ps.publish(msg); err != nil {
		ps.logger.Warn(fmt.Sprintf("Publishing failed: %s", err))
		return
	}
}

func (ps pubsub) publish(msg mainflux.RawMessage) error {
	output := mainflux.OutputSenML
	normalized, err := ps.svc.Normalize(msg)
	if err != nil {
		switch ct := msg.ContentType; ct {
		case senML:
			return err
		case "":
			output = outputUnknown
		default:
			output = fmt.Sprintf("out.%s", ct)
		}

		if err := ps.nc.Publish(output, msg.GetPayload()); err != nil {
			ps.logger.Warn(fmt.Sprintf("Publishing failed: %s", err))
			return err
		}
	}

	for _, v := range normalized.Messages {
		data, err := proto.Marshal(&v)
		if err != nil {
			ps.logger.Warn(fmt.Sprintf("Marshalling failed: %s", err))
			return err
		}

		if err := ps.nc.Publish(output, data); err != nil {
			ps.logger.Warn(fmt.Sprintf("Publishing failed: %s", err))
			return err
		}
	}

	return nil
}
