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

func (re *re) save(original *messaging.Message) lua.LGFunction {
	return func(l *lua.LState) int {
		ls := l.ToString(1)
		var message senml.Message

		if err := json.Unmarshal([]byte(ls), &message); err != nil {
			return 0
		}

		payload, err := json.Marshal(message)
		if err != nil {
			return 0
		}

		ctx := context.Background()
		m := &messaging.Message{
			Domain:    original.Domain,
			Publisher: original.Publisher,
			Created:   original.Created,
			Channel:   original.Channel,
			Subtopic:  original.Subtopic,
			Protocol:  original.Protocol,
			Payload:   payload,
		}

		if err := re.writersPub.Publish(ctx, message.Channel, m); err != nil {
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
