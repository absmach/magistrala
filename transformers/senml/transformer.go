// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package senml

import (
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/messaging"
	"github.com/mainflux/mainflux/transformers"
	"github.com/mainflux/senml"
)

const (
	// JSON represents SenML in JSON format content type.
	JSON = "application/senml+json"
	// CBOR represents SenML in CBOR format content type.
	CBOR = "application/senml+cbor"
)

var (
	errDecode    = errors.New("failed to decode senml")
	errNormalize = errors.New("faled to normalize senml")
)

var formats = map[string]senml.Format{
	JSON: senml.JSON,
	CBOR: senml.CBOR,
}

type transformer struct {
	format senml.Format
}

// New returns transformer service implementation for SenML messages.
func New(contentFormat string) transformers.Transformer {
	format, ok := formats[contentFormat]
	if !ok {
		format = formats[JSON]
	}

	return transformer{
		format: format,
	}
}

func (t transformer) Transform(msg messaging.Message) (interface{}, error) {
	raw, err := senml.Decode(msg.Payload, t.format)
	if err != nil {
		return nil, errors.Wrap(errDecode, err)
	}

	normalized, err := senml.Normalize(raw)
	if err != nil {
		return nil, errors.Wrap(errNormalize, err)
	}

	msgs := make([]Message, len(normalized.Records))
	for i, v := range normalized.Records {
		// Use reception timestamp if SenML messsage Time is missing
		t := v.Time
		if t == 0 {
			// Convert the Unix timestamp in nanoseconds to float64
			t = float64(msg.Created) / float64(1e9)
		}

		msgs[i] = Message{
			Channel:     msg.Channel,
			Subtopic:    msg.Subtopic,
			Publisher:   msg.Publisher,
			Protocol:    msg.Protocol,
			Name:        v.Name,
			Unit:        v.Unit,
			Time:        t,
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
