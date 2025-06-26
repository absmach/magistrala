// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"context"
	"encoding/json"

	"github.com/absmach/supermq/pkg/messaging"
)

const protocol = "nats"

type ChannelPublisher struct {
	RePubSub messaging.PubSub `json:"-"`
	Channel  string           `json:"channel"`
	Topic    string           `json:"topic"`
}

func (p *ChannelPublisher) Run(ctx context.Context, msg *messaging.Message, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   p.Channel,
		Subtopic:  p.Topic,
		Protocol:  protocol,
		Payload:   data,
	}

	topic := messaging.EncodeTopicSuffix(msg.Domain, p.Channel, p.Topic)
	if err := p.RePubSub.Publish(ctx, topic, m); err != nil {
		return err
	}

	return nil
}

func (cp *ChannelPublisher) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type":    ChannelsType.String(),
		"channel": cp.Channel,
		"topic":   cp.Topic,
	})
}
