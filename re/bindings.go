// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"context"
	"encoding/json"

	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/transformers/senml"
	lua "github.com/yuin/gopher-lua"
)

func (re *re) save(ctx context.Context, original *messaging.Message) lua.LGFunction {
	return func(l *lua.LState) int {
		table := l.ToTable(1)
		val := convertLua(table)
		// In case there is a single SenML value, convert to slice so we can unmarshal.
		if _, ok := val.([]any); !ok {
			val = []any{val}
		}
		data, err := json.Marshal(val)
		if err != nil {
			return 0
		}

		var message []senml.Message
		if err := json.Unmarshal(data, &message); err != nil {
			return 0
		}

		m := &messaging.Message{
			Domain:    original.Domain,
			Publisher: original.Publisher,
			Created:   original.Created,
			Channel:   original.Channel,
			Subtopic:  original.Subtopic,
			Protocol:  original.Protocol,
			Payload:   data,
		}
		if err := re.writersPub.Publish(ctx, original.Channel, m); err != nil {
			return 0
		}
		return 1
	}
}

func (re *re) sendEmail(L *lua.LState) int {
	recipientsTable := L.ToTable(1)
	subject := L.ToString(2)
	content := L.ToString(3)

	var recipients []string
	recipientsTable.ForEach(func(_, value lua.LValue) {
		if str, ok := value.(lua.LString); ok {
			recipients = append(recipients, string(str))
		}
	})

	if err := re.email.SendEmailNotification(recipients, "", subject, "", "", content, ""); err != nil {
		return 0
	}
	return 1
}
