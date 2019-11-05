// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml

import (
	"github.com/cisco/senml"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/transformers"
)

var formats = map[string]senml.Format{
	SenMLJSON: senml.JSON,
	SenMLCBOR: senml.CBOR,
}

type transformer struct{}

// New returns transformer service implementation for SenML messages.
func New() transformers.Transformer {
	return transformer{}
}

func (n transformer) Transform(msg mainflux.Message) (interface{}, error) {
	format, ok := formats[msg.ContentType]
	if !ok {
		format = senml.JSON
	}

	raw, err := senml.Decode(msg.Payload, format)
	if err != nil {
		return nil, err
	}

	normalized := senml.Normalize(raw)

	msgs := make([]Message, len(normalized.Records))
	for k, v := range normalized.Records {
		m := Message{
			Channel:    msg.Channel,
			Subtopic:   msg.Subtopic,
			Publisher:  msg.Publisher,
			Protocol:   msg.Protocol,
			Name:       v.Name,
			Unit:       v.Unit,
			Time:       v.Time,
			UpdateTime: v.UpdateTime,
			Link:       v.Link,
			Sum:        v.Sum,
		}

		switch {
		case v.Value != nil:
			m.Value = v.Value
		case v.BoolValue != nil:
			m.BoolValue = v.BoolValue
		case v.DataValue != "":
			m.DataValue = &v.DataValue
		case v.StringValue != "":
			m.StringValue = &v.StringValue
		}

		msgs[k] = m
	}

	return msgs, nil
}
