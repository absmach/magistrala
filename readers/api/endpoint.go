//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
)

func listMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(listMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		messages := svc.ReadAll(req.chanID, req.offset, req.limit)

		msgs := []message{}
		for _, m := range messages {
			msg := message{
				Channel:    m.Channel,
				Publisher:  m.Publisher,
				Protocol:   m.Protocol,
				Name:       m.Name,
				Unit:       m.Unit,
				Time:       m.Time,
				UpdateTime: m.UpdateTime,
				Link:       m.Link,
			}

			switch m.Value.(type) {
			case *mainflux.Message_FloatValue:
				val := m.GetFloatValue()
				msg.Value = &val
			case *mainflux.Message_StringValue:
				strVal := m.GetStringValue()
				msg.StringValue = &strVal
			case *mainflux.Message_DataValue:
				dataVal := m.GetDataValue()
				msg.DataValue = &dataVal
			case *mainflux.Message_BoolValue:
				boolVal := m.GetBoolValue()
				msg.BoolValue = &boolVal
			}

			if m.GetValueSum() != nil {
				valueSum := m.GetValueSum().Value
				msg.ValueSum = &valueSum
			}

			msgs = append(msgs, msg)
		}
		return listMessagesRes{Messages: msgs}, nil
	}
}
