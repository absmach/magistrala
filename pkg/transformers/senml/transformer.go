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

	maxRelativeTime = 1 << 28
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

func (t transformer) Transform(msg *messaging.Message) (any, error) {
	raw, err := senml.Decode(msg.GetPayload(), t.format)
	if err != nil {
		return nil, errors.Wrap(errDecode, err)
	}

	// Accumulate bn forward across the raw pack (RFC 8428 §4.6), before
	// Normalize folds it into Name and discards it. Stashed in the otherwise
	// unused Link field so it survives Normalize's internal sort — Swap moves
	// whole records, so the accumulated value travels with the record it
	// belongs to regardless of how Time ties are broken. Link is never
	// mapped to the output Message, so this does not change any existing
	// behaviour.
	//
	// Seeded from the broker-level device_id so a pack with no bn in any
	// record still carries it, matching the JSON transformer's handling of
	// the same field.
	deviceID := msg.GetDeviceId()
	for i := range raw.Records {
		if raw.Records[i].BaseName != "" {
			deviceID = raw.Records[i].BaseName
		}
		raw.Records[i].Link = deviceID
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
			t = float64(msg.GetCreated())
		}

		// If time is below 2**28 it is relative to the current time
		// https://datatracker.ietf.org/doc/html/rfc8428#section-4.5.3
		if t >= maxRelativeTime {
			t = transformers.ToUnixNano(t)
		}
		if v.UpdateTime >= maxRelativeTime {
			v.UpdateTime = transformers.ToUnixNano(v.UpdateTime)
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
			DeviceId:    v.Link,
		}
	}

	return msgs, nil
}
