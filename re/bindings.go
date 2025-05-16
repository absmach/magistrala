// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/senml"
	"github.com/absmach/supermq/pkg/messaging"
	lua "github.com/yuin/gopher-lua"
)

func (re *re) sendEmail(l *lua.LState) int {
	recipientsTable := l.ToTable(1)
	subject := l.ToString(2)
	content := l.ToString(3)

	var recipients []string
	recipientsTable.ForEach(func(_, value lua.LValue) {
		if str, ok := value.(lua.LString); ok {
			recipients = append(recipients, string(str))
		}
	})

	if err := re.email.SendEmailNotification(recipients, "", subject, "", "", content, "", make(map[string][]byte)); err != nil {
		return 0
	}
	return 1
}

func (re *re) sendAlarm(ctx context.Context, ruleID string, original *messaging.Message) lua.LGFunction {
	return func(l *lua.LState) int {
		processAlarm := func(alarmTable *lua.LTable) int {
			val := convertLua(alarmTable)
			data, err := json.Marshal(val)
			if err != nil {
				return 0
			}

			alarm := alarms.Alarm{
				RuleID:    ruleID,
				DomainID:  original.Domain,
				ClientID:  original.Publisher,
				ChannelID: original.Channel,
				Subtopic:  original.Subtopic,
			}
			if err := json.Unmarshal(data, &alarm); err != nil {
				return 0
			}

			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(alarm); err != nil {
				return 0
			}

			m := &messaging.Message{
				Domain:    original.Domain,
				Publisher: original.Publisher,
				Created:   original.Created,
				Channel:   original.Channel,
				Subtopic:  original.Subtopic,
				Protocol:  original.Protocol,
				Payload:   buf.Bytes(),
			}

			if err := re.alarmsPub.Publish(ctx, original.Channel, m); err != nil {
				return 0
			}
			return 1
		}
		table := l.ToTable(1)
		if table.RawGetInt(1) != lua.LNil {
			table.ForEach(func(_, value lua.LValue) {
				if alarmTable, ok := value.(*lua.LTable); ok {
					processAlarm(alarmTable)
				}
			})
		} else {
			processAlarm(table)
		}

		return 1
	}
}

func (re *re) saveSenml(ctx context.Context, val interface{}, msg *messaging.Message) error {
	// In case there is a single SenML value, convert to slice so we can decode.
	if _, ok := val.([]any); !ok {
		val = []any{val}
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	if _, err := senml.Decode(data, senml.JSON); err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   msg.Channel,
		Subtopic:  msg.Subtopic,
		Protocol:  msg.Protocol,
		Payload:   data,
	}
	if err := re.writersPub.Publish(ctx, msg.Channel, m); err != nil {
		return err
	}
	return nil
}

func (re *re) publishChannel(ctx context.Context, val interface{}, channel, subtopic string, msg *messaging.Message) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   channel,
		Subtopic:  subtopic,
		Protocol:  protocol,
		Payload:   data,
	}
	if err := re.rePubSub.Publish(ctx, channel, m); err != nil {
		return err
	}
	return nil
}
