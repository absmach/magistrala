// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/senml"
	"github.com/absmach/supermq/pkg/messaging"
	lua "github.com/yuin/gopher-lua"
)

func luaEncrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	enc, err := encrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to encrypt: %v", err)))
		return 2
	}
	l.Push(lua.LString(hex.EncodeToString(enc)))

	return 1
}

func luaDecrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	dec, err := decrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decrypt: %v", err)))
		return 2
	}

	l.Push(lua.LString(hex.EncodeToString(dec)))

	return 1
}

func decodeParams(l *lua.LState) (key, iv, data []byte, err error) {
	keyStr := l.ToString(1)
	ivStr := l.ToString(2)
	dataStr := l.ToString(3)

	key, err = hex.DecodeString(keyStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode key: %v", err)
	}

	iv, err = hex.DecodeString(ivStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode IV: %v", err)
	}

	data, err = hex.DecodeString(dataStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode data: %v", err)
	}

	return key, iv, data, nil
}

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
