// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
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
		ruleID, measurement, value, unit, cause, domainID string
		severity                                          uint8
	)

	for _, msg := range msgs {
		if msg.Name == "rule_id" {
			if msg.StringValue != nil {
				ruleID = *msg.StringValue
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
		if msg.Name == "domain_id" {
			if msg.StringValue != nil {
				domainID = *msg.StringValue
			}
		}
	}

	a := alarms.Alarm{
		RuleID:      ruleID,
		Status:      alarms.ReportedStatus,
		Measurement: measurement,
		Value:       value,
		Unit:        unit,
		Cause:       cause,
		Severity:    severity,
		DomainID:    domainID,
		CreatedAt:   time.Now(),
		Metadata: alarms.Metadata{
			"payload": msgs,
		},
	}

	if err := a.Validate(); err != nil {
		return err
	}

	return c.svc.CreateAlarm(ctx, a)
}

func (c consumer) saveJSON(ctx context.Context, msgs smqjson.Messages) error {
	var (
		ruleID, measurement, value, unit, cause, domainID string
		severity                                          uint8
	)

	for _, msg := range msgs.Data {
		if getString(msg.Payload, "rule_id") != "" {
			ruleID = getString(msg.Payload, "rule_id")
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
		if getString(msg.Payload, "cause") != "" {
			cause = getString(msg.Payload, "cause")
		}
		if getUint8(msg.Payload, "severity") != nil {
			s := getUint8(msg.Payload, "severity")
			severity = *s
		}
		if getString(msg.Payload, "domain_id") != "" {
			domainID = getString(msg.Payload, "domain_id")
		}
	}

	a := alarms.Alarm{
		RuleID:      ruleID,
		Status:      alarms.ReportedStatus,
		Measurement: measurement,
		Value:       value,
		Unit:        unit,
		Cause:       cause,
		Severity:    severity,
		DomainID:    domainID,
		CreatedAt:   time.Now(),
		Metadata: alarms.Metadata{
			"payload": msgs,
		},
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
