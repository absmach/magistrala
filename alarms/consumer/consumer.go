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
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
)

type handler struct {
	svc    alarms.Service
	logger *slog.Logger
}

func Newhandler(svc alarms.Service, logger *slog.Logger) messaging.MessageHandler {
	return &handler{svc: svc, logger: logger}
}

func (h handler) Handle(msg *messaging.Message) (err error) {
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
	alarm.ClientID = msg.GetPublisher()
	alarm.Subtopic = msg.GetSubtopic()

	if alarm.CreatedAt.IsZero() {
		alarm.CreatedAt = time.Unix(0, int64(msg.GetCreated()))
	}

	if err := alarm.Validate(); err != nil {
		return err
	}

	return h.svc.CreateAlarm(context.Background(), alarm)
}

func (h handler) Cancel() error {
	return nil
}
