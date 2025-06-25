// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/pkg/messaging"
)

type Alarm struct {
	AlarmsPub messaging.Publisher
	RuleID    string `json:"rule_id"`
}

func (a *Alarm) Run(ctx context.Context, msg *messaging.Message, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	alarm := alarms.Alarm{
		RuleID:    a.RuleID,
		DomainID:  msg.Domain,
		ClientID:  msg.Publisher,
		ChannelID: msg.Channel,
		Subtopic:  msg.Subtopic,
	}
	if err := json.Unmarshal(data, &alarm); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(alarm); err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   msg.Channel,
		Subtopic:  msg.Subtopic,
		Protocol:  msg.Protocol,
		Payload:   buf.Bytes(),
	}

	topic := messaging.EncodeMessageTopic(msg)
	if err := a.AlarmsPub.Publish(ctx, topic, m); err != nil {
		return err
	}
	return nil
}
