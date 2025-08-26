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
	AlarmsPub messaging.Publisher `json:"-"`
	RuleID    string              `json:"rule_id"`
}

func (a *Alarm) Run(ctx context.Context, msg *messaging.Message, val any) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	var alarmsList []alarms.Alarm
	if err := json.Unmarshal(data, &alarmsList); err != nil {
		var single alarms.Alarm
		if err := json.Unmarshal(data, &single); err != nil {
			return err
		}
		alarmsList = []alarms.Alarm{single}
	}

	for _, alarm := range alarmsList {
		if err := a.processAlarm(ctx, msg, alarm); err != nil {
			return err
		}
	}

	return nil
}

func (a *Alarm) processAlarm(ctx context.Context, msg *messaging.Message, alarm alarms.Alarm) error {
	alarm.RuleID = a.RuleID
	alarm.DomainID = msg.Domain
	alarm.ClientID = msg.Publisher
	alarm.ChannelID = msg.Channel
	alarm.Subtopic = msg.Subtopic

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

func (a *Alarm) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": AlarmsType.String(),
	})
}
