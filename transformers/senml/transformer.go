// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml

import (
	"github.com/mainflux/mainflux/broker"
	"github.com/mainflux/mainflux/transformers"
	"github.com/mainflux/senml"
)

var formats = map[string]senml.Format{
	JSON: senml.JSON,
	CBOR: senml.CBOR,
}

type transformer struct{}

// New returns transformer service implementation for SenML messages.
func New() transformers.Transformer {
	return transformer{}
}

func (n transformer) Transform(msg broker.Message) (interface{}, error) {
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		return nil, err
	}

	normalized, err := senml.Normalize(raw)
	if err != nil {
		return nil, err
	}

	msgs := make([]Message, len(normalized.Records))
	for i, v := range normalized.Records {
		msgs[i] = Message{
			Channel:     msg.Channel,
			Subtopic:    msg.Subtopic,
			Publisher:   msg.Publisher,
			Protocol:    msg.Protocol,
			Name:        v.Name,
			Unit:        v.Unit,
			Time:        v.Time,
			UpdateTime:  v.UpdateTime,
			Value:       v.Value,
			BoolValue:   v.BoolValue,
			DataValue:   v.DataValue,
			StringValue: v.StringValue,
			Sum:         v.Sum,
		}
	}

	return msgs, nil
}
