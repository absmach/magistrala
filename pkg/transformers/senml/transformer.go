// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package senml

import (
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/transformers"
	"github.com/absmach/senml"
)

const (
	// JSON represents SenML in JSON format content type.
	JSON = "application/senml+json"
	// CBOR represents SenML in CBOR format content type.
	CBOR = "application/senml+cbor"
)

var (
	errDecode    = errors.New("failed to decode senml")
	errNormalize = errors.New("failed to normalize senml")
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

func (t transformer) Transform(msg *messaging.Message) (interface{}, error) {
	raw, err := senml.Decode(msg.GetPayload(), t.format)
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
			t = float64(msg.GetCreated()) / float64(1e9)
		}

		msgs[i] = Message{
			Channel:     msg.GetChannel(),
			Subtopic:    msg.GetSubtopic(),
			Publisher:   msg.GetPublisher(),
			Protocol:    msg.GetProtocol(),
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
