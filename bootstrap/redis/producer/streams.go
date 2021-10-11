// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/bootstrap"
)

const (
	streamID  = "mainflux.bootstrap"
	streamLen = 1000
)

var _ bootstrap.Service = (*eventStore)(nil)

type eventStore struct {
	svc    bootstrap.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around bootstrap service that sends
// events to event store.
func NewEventStoreMiddleware(svc bootstrap.Service, client *redis.Client) bootstrap.Service {
	return eventStore{
		svc:    svc,
		client: client,
	}
}

func (es eventStore) Add(ctx context.Context, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	saved, err := es.svc.Add(ctx, token, cfg)
	if err != nil {
		return saved, err
	}

	var channels []string
	for _, ch := range saved.MFChannels {
		channels = append(channels, ch.ID)
	}

	ev := createConfigEvent{
		mfThing:    saved.MFThing,
		owner:      saved.Owner,
		name:       saved.Name,
		mfChannels: channels,
		externalID: saved.ExternalID,
		content:    saved.Content,
		timestamp:  time.Now(),
	}

	es.add(ctx, ev)

	return saved, err
}

func (es eventStore) View(ctx context.Context, token, id string) (bootstrap.Config, error) {
	return es.svc.View(ctx, token, id)
}

func (es eventStore) Update(ctx context.Context, token string, cfg bootstrap.Config) error {
	if err := es.svc.Update(ctx, token, cfg); err != nil {
		return err
	}

	ev := updateConfigEvent{
		mfThing:   cfg.MFThing,
		name:      cfg.Name,
		content:   cfg.Content,
		timestamp: time.Now(),
	}

	es.add(ctx, ev)

	return nil
}

func (es eventStore) UpdateCert(ctx context.Context, token, thingKey, clientCert, clientKey, caCert string) error {
	return es.svc.UpdateCert(ctx, token, thingKey, clientCert, clientKey, caCert)
}

func (es eventStore) UpdateConnections(ctx context.Context, token, id string, connections []string) error {
	if err := es.svc.UpdateConnections(ctx, token, id, connections); err != nil {
		return err
	}

	ev := updateConnectionsEvent{
		mfThing:    id,
		mfChannels: connections,
		timestamp:  time.Now(),
	}

	es.add(ctx, ev)

	return nil
}

func (es eventStore) List(ctx context.Context, token string, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	return es.svc.List(ctx, token, filter, offset, limit)
}

func (es eventStore) Remove(ctx context.Context, token, id string) error {
	if err := es.svc.Remove(ctx, token, id); err != nil {
		return err
	}

	ev := removeConfigEvent{
		mfThing:   id,
		timestamp: time.Now(),
	}

	es.add(ctx, ev)

	return nil
}

func (es eventStore) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	cfg, err := es.svc.Bootstrap(ctx, externalKey, externalID, secure)

	ev := bootstrapEvent{
		externalID: externalID,
		timestamp:  time.Now(),
		success:    true,
	}

	if err != nil {
		ev.success = false
	}

	es.add(ctx, ev)

	return cfg, err
}

func (es eventStore) ChangeState(ctx context.Context, token, id string, state bootstrap.State) error {
	if err := es.svc.ChangeState(ctx, token, id, state); err != nil {
		return err
	}

	ev := changeStateEvent{
		mfThing:   id,
		state:     state,
		timestamp: time.Now(),
	}

	es.add(ctx, ev)

	return nil
}

func (es eventStore) RemoveConfigHandler(ctx context.Context, id string) error {
	return es.svc.RemoveConfigHandler(ctx, id)
}

func (es eventStore) RemoveChannelHandler(ctx context.Context, id string) error {
	return es.svc.RemoveChannelHandler(ctx, id)
}

func (es eventStore) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) error {
	return es.svc.UpdateChannelHandler(ctx, channel)
}

func (es eventStore) DisconnectThingHandler(ctx context.Context, channelID, thingID string) error {
	return es.svc.DisconnectThingHandler(ctx, channelID, thingID)
}

func (es eventStore) add(ctx context.Context, ev event) error {
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       ev.encode(),
	}

	return es.client.XAdd(ctx, record).Err()
}
