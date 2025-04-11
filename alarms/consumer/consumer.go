// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"bytes"
	"context"
	"encoding/gob"
	"log/slog"
	"time"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/supermq/consumers"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
)

var _ consumers.BlockingConsumer = (*consumer)(nil)

type consumer struct {
	svc    alarms.Service
	logger *slog.Logger
}

func NewConsumer(svc alarms.Service, logger *slog.Logger) consumers.BlockingConsumer {
	return &consumer{svc: svc, logger: logger}
}

func (c consumer) ConsumeBlocking(ctx context.Context, message interface{}) (err error) {
	switch m := message.(type) {
	case *messaging.Message:
		return c.handleMessage(ctx, m)
	default:
		c.logger.Warn("Invalid message received")

		return nil
	}
}

func (c consumer) handleMessage(ctx context.Context, msg *messaging.Message) (err error) {
	if msg == nil {
		return errors.New("message is empty")
	}
	if msg.GetPayload() == nil {
		return errors.New("message payload is empty")
	}

	var alarm alarms.Alarm
	if err := gob.NewDecoder(bytes.NewReader(msg.GetPayload())).Decode(&alarm); err != nil {
		return err
	}
	alarm.DomainID = msg.GetDomain()
	alarm.ChannelID = msg.GetChannel()
	alarm.ThingID = msg.GetPublisher()
	alarm.Subtopic = msg.GetSubtopic()

	if alarm.CreatedAt.IsZero() {
		alarm.CreatedAt = time.Unix(0, int64(msg.GetCreated()))
	}

	if err := alarm.Validate(); err != nil {
		return err
	}

	return c.svc.CreateAlarm(ctx, alarm)
}
