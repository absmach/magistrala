// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"context"
	"encoding/json"

	"github.com/absmach/senml"
	"github.com/absmach/supermq/pkg/messaging"
)

type SenML struct {
	WritersPub messaging.Publisher `json:"-"`
}

func (s *SenML) Run(ctx context.Context, msg *messaging.Message, val interface{}) error {
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
	topic := messaging.EncodeMessageTopic(msg)
	if err := s.WritersPub.Publish(ctx, topic, m); err != nil {
		return err
	}

	return nil
}

func (senml *SenML) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"type": SaveSenMLType.String(),
	})
}
