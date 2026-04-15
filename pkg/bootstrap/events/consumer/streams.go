// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/bootstrap"
	mgerrors "github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
)

const (
	stream = "events.magistrala.*.*"

	clientRemove = "client.remove"

	channelConnect    = "channel.connect"
	channelDisconnect = "channel.disconnect"
)

type eventHandler struct {
	svc bootstrap.Service
}

// BootstrapEventsSubscribe subscribes bootstrap config-state handlers to the event store.
func BootstrapEventsSubscribe(ctx context.Context, svc bootstrap.Service, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, "bootstrap-es-sub", logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:   stream,
		Consumer: esConsumerName,
		Handler:  NewEventHandler(svc),
		Ordered:  true,
	}
	return subscriber.Subscribe(ctx, subConfig)
}

// NewEventHandler returns bootstrap events handler.
func NewEventHandler(svc bootstrap.Service) events.EventHandler {
	return &eventHandler{
		svc: svc,
	}
}

func (es *eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	op, _ := msg["operation"].(string)
	switch op {
	case clientRemove:
		return es.removeConfigHandler(ctx, msg)
	case channelConnect:
		return es.connectHandler(ctx, decodeConnection(msg))
	case channelDisconnect:
		return es.disconnectHandler(ctx, decodeConnection(msg))
	}

	return nil
}

func (es *eventHandler) removeConfigHandler(ctx context.Context, data map[string]any) error {
	id := readString(data, "id")
	if id == "" {
		return svcerr.ErrMalformedEntity
	}
	if err := es.svc.RemoveConfigHandler(ctx, id); err != nil && !mgerrors.Contains(err, repoerr.ErrNotFound) {
		return err
	}
	return nil
}

func (es *eventHandler) connectHandler(ctx context.Context, ce connectionEvent) error {
	if len(ce.channelIDs) == 0 || len(ce.clientIDs) == 0 {
		return svcerr.ErrMalformedEntity
	}

	for _, channelID := range ce.channelIDs {
		for _, clientID := range ce.clientIDs {
			if err := es.svc.ConnectClientHandler(ctx, channelID, clientID); err != nil && !mgerrors.Contains(err, repoerr.ErrNotFound) {
				return err
			}
		}
	}
	return nil
}

func (es *eventHandler) disconnectHandler(ctx context.Context, ce connectionEvent) error {
	if len(ce.channelIDs) == 0 || len(ce.clientIDs) == 0 {
		return svcerr.ErrMalformedEntity
	}

	for _, channelID := range ce.channelIDs {
		for _, clientID := range ce.clientIDs {
			if err := es.svc.DisconnectClientHandler(ctx, channelID, clientID); err != nil && !mgerrors.Contains(err, repoerr.ErrNotFound) {
				return err
			}
		}
	}
	return nil
}
