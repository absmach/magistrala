// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/errors"
	smqjson "github.com/absmach/supermq/pkg/transformers/json"
	"github.com/absmach/supermq/pkg/transformers/senml"
)

var _ consumers.BlockingConsumer = (*consumer)(nil)

type consumer struct {
	svc alarms.Service
}

func NewConsumer(svc alarms.Service) consumers.BlockingConsumer {
	return &consumer{svc: svc}
}

func (c consumer) ConsumeBlocking(ctx context.Context, message interface{}) (err error) {
	switch m := message.(type) {
	case smqjson.Messages:
		return c.saveJSON(ctx, m)
	default:
		return c.saveSenml(ctx, m)
	}
}

func (c consumer) saveSenml(ctx context.Context, messages interface{}) (err error) {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errors.New("invalid message representation")
	}

	if len(msgs) == 0 {
		return errors.New("message is empty")
	}

	var (
		ruleID, domainID, measurement, value, unit, threshold, cause, assigneeID string
		severity                                                                 uint8
		createdAt                                                                float64
		metadata                                                                 map[string]interface{}
	)

	for _, msg := range msgs {
		if msg.Name == "rule_id" {
			if msg.StringValue != nil {
				ruleID = *msg.StringValue
			}
		}
		if msg.Name == "domain_id" {
			if msg.StringValue != nil {
				domainID = *msg.StringValue
			}
		}
		if msg.Name == "measurement" {
			if msg.StringValue != nil {
				measurement = *msg.StringValue
			}
		}
		if msg.Name == "value" {
			if msg.Value != nil {
				value = strconv.FormatFloat(*msg.Value, 'f', 2, 64)
			}
		}
		if msg.Name == "unit" {
			if msg.StringValue != nil {
				unit = *msg.StringValue
			}
		}
		if msg.Name == "threshold" {
			if msg.Value != nil {
				threshold = strconv.FormatFloat(*msg.Value, 'f', 2, 64)
			}
		}
		if msg.Name == "cause" {
			if msg.StringValue != nil {
				cause = *msg.StringValue
			}
		}
		if msg.Name == "severity" {
			if msg.Value != nil {
				severity = uint8(*msg.Value)
			}
		}
		if msg.Name == "assignee_id" {
			if msg.StringValue != nil {
				assigneeID = *msg.StringValue
			}
		}
		if msg.Name == "created_at" {
			if msg.Value != nil {
				createdAt = *msg.Value
			}
		}
		if msg.Name == "metadata" {
			if msg.DataValue != nil {
				data, err := json.Marshal(msg.DataValue)
				if err == nil {
					var payload map[string]interface{}
					if err := json.Unmarshal(data, &payload); err == nil {
						metadata = payload
					}
				}
			}
		}
	}
	if createdAt == 0 {
		createdAt = msgs[0].Time
	}

	a := alarms.Alarm{
		RuleID:      ruleID,
		DomainID:    domainID,
		ChannelID:   msgs[0].Channel,
		ThingID:     msgs[0].Publisher,
		Subtopic:    msgs[0].Subtopic,
		Status:      alarms.ActiveStatus,
		Measurement: measurement,
		Value:       value,
		Unit:        unit,
		Threshold:   threshold,
		Cause:       cause,
		Severity:    severity,
		AssigneeID:  assigneeID,
		CreatedAt:   time.Unix(0, int64(createdAt)),
		Metadata:    metadata,
	}

	if err := a.Validate(); err != nil {
		return err
	}

	return c.svc.CreateAlarm(ctx, a)
}

func (c consumer) saveJSON(ctx context.Context, msgs smqjson.Messages) error {
	var (
		ruleID, domainID, measurement, value, unit, threshold, cause, assigneeID string
		severity                                                                 uint8
		createdAt                                                                float64
		metadata                                                                 map[string]interface{}
	)

	for _, msg := range msgs.Data {
		if getString(msg.Payload, "rule_id") != "" {
			ruleID = getString(msg.Payload, "rule_id")
		}
		if getString(msg.Payload, "domain_id") != "" {
			domainID = getString(msg.Payload, "domain_id")
		}
		if getString(msg.Payload, "measurement") != "" {
			measurement = getString(msg.Payload, "measurement")
		}
		if getString(msg.Payload, "value") != "" {
			value = getString(msg.Payload, "value")
		}
		if getString(msg.Payload, "unit") != "" {
			unit = getString(msg.Payload, "unit")
		}
		if getString(msg.Payload, "threshold") != "" {
			threshold = getString(msg.Payload, "threshold")
		}
		if getString(msg.Payload, "cause") != "" {
			cause = getString(msg.Payload, "cause")
		}
		if getUint8(msg.Payload, "severity") != nil {
			s := getUint8(msg.Payload, "severity")
			severity = *s
		}
		if getString(msg.Payload, "assignee_id") != "" {
			assigneeID = getString(msg.Payload, "assignee_id")
		}
		if getFloat64(msg.Payload, "created_at") != 0 {
			createdAt = getFloat64(msg.Payload, "created_at")
		}
		if getString(msg.Payload, "metadata") != "" {
			data, err := json.Marshal(getString(msg.Payload, "metadata"))
			if err == nil {
				var payload map[string]interface{}
				if err := json.Unmarshal(data, &payload); err == nil {
					metadata = payload
				}
			}
		}
	}
	if createdAt == 0 {
		createdAt = float64(msgs.Data[0].Created)
	}

	a := alarms.Alarm{
		RuleID:      ruleID,
		DomainID:    domainID,
		ChannelID:   msgs.Data[0].Channel,
		ThingID:     msgs.Data[0].Publisher,
		Subtopic:    msgs.Data[0].Subtopic,
		Status:      alarms.ActiveStatus,
		Measurement: measurement,
		Value:       value,
		Unit:        unit,
		Threshold:   threshold,
		Cause:       cause,
		Severity:    severity,
		AssigneeID:  assigneeID,
		CreatedAt:   time.Unix(0, int64(createdAt)),
		Metadata:    metadata,
	}

	if err := a.Validate(); err != nil {
		return err
	}

	return c.svc.CreateAlarm(ctx, a)
}

func getString(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func getUint8(payload map[string]interface{}, key string) *uint8 {
	if payload == nil {
		return nil
	}
	value, ok := payload[key]
	if !ok {
		return nil
	}
	ui, ok := value.(uint8)
	if ok {
		return &ui
	}
	val, ok := value.(uint16)
	if ok {
		v := uint8(val)
		return &v
	}
	f, ok := value.(float64)
	if ok {
		v := uint8(f)
		return &v
	}
	i, ok := value.(int)
	if ok {
		v := uint8(i)
		return &v
	}
	v := uint8(0)
	return &v
}

func getFloat64(payload map[string]interface{}, key string) float64 {
	if payload == nil {
		return 0
	}
	value, ok := payload[key]
	if !ok {
		return 0
	}
	f, ok := value.(float64)
	if ok {
		return f
	}
	val, ok := value.(uint64)
	if ok {
		v := float64(val)
		return v
	}
	i, ok := value.(int)
	if ok {
		v := float64(i)
		return v
	}

	return 0
}
