//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package normalizer

import (
	"strings"

	"github.com/cisco/senml"
	"github.com/mainflux/mainflux"
)

type normalizer struct{}

// New returns normalizer service implementation.
func New() Service {
	return normalizer{}
}

func (n normalizer) Normalize(msg mainflux.RawMessage) (NormalizedData, error) {
	raw, err := senml.Decode(msg.Payload, senml.JSON)
	if err != nil {
		return NormalizedData{}, err
	}

	normalized := senml.Normalize(raw)

	msgs := make([]mainflux.Message, len(normalized.Records))
	for k, v := range normalized.Records {
		m := mainflux.Message{
			Channel:    msg.Channel,
			Publisher:  msg.Publisher,
			Protocol:   msg.Protocol,
			Name:       v.Name,
			Unit:       v.Unit,
			Time:       v.Time,
			UpdateTime: v.UpdateTime,
			Link:       v.Link,
		}

		switch {
		case v.Value != nil:
			m.Value = &mainflux.Message_FloatValue{FloatValue: *v.Value}
		case v.BoolValue != nil:
			m.Value = &mainflux.Message_BoolValue{BoolValue: *v.BoolValue}
		case v.DataValue != "":
			m.Value = &mainflux.Message_DataValue{DataValue: v.DataValue}
		case v.StringValue != "":
			m.Value = &mainflux.Message_StringValue{StringValue: v.StringValue}
		}

		if v.Sum != nil {
			m.ValueSum = &mainflux.SumValue{Value: *v.Sum}
		}

		msgs[k] = m
	}

	output := strings.ToLower(msg.ContentType)

	return NormalizedData{
		ContentType: output,
		Messages:    msgs,
	}, nil
}
