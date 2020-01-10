// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/mainflux/twins/mqtt"
	"github.com/nats-io/go-nats"
)

const (
	queue = "twins"
	input = "channel.>"
)

var crudOp = map[string]string{
	"stateSucc": "state/success",
	"stateFail": "state/failure",
}

type pubsub struct {
	natsClient *nats.Conn
	mqttClient mqtt.Mqtt
	logger     log.Logger
	svc        twins.Service
}

// Subscribe to appropriate NATS topic
func Subscribe(nc *nats.Conn, mc mqtt.Mqtt, svc twins.Service, logger log.Logger) {
	ps := pubsub{
		natsClient: nc,
		mqttClient: mc,
		logger:     logger,
		svc:        svc,
	}
	ps.natsClient.QueueSubscribe(input, queue, ps.handleMsg)
}

func (ps pubsub) handleMsg(m *nats.Msg) {
	var msg mainflux.Message
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		ps.logger.Warn(fmt.Sprintf("Unmarshalling failed: %s", err))
		return
	}

	if msg.Channel == ps.mqttClient.Channel() {
		return
	}

	ps.svc.SaveState(&msg)
}
