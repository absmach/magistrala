// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
)

var _ bootstrap.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc bootstrap.Service
}

// NewEventStoreMiddleware returns wrapper around bootstrap service that sends
// events to event store.
func NewEventStoreMiddleware(svc bootstrap.Service, publisher events.Publisher) bootstrap.Service {
	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}
}

func (es *eventStore) Add(ctx context.Context, session mgauthn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	saved, err := es.svc.Add(ctx, session, token, cfg)
	if err != nil {
		return saved, err
	}

	ev := configEvent{
		saved, configCreate,
	}

	if err := es.Publish(ctx, ev); err != nil {
		return saved, err
	}

	return saved, err
}

func (es *eventStore) View(ctx context.Context, session mgauthn.Session, id string) (bootstrap.Config, error) {
	cfg, err := es.svc.View(ctx, session, id)
	if err != nil {
		return cfg, err
	}
	ev := configEvent{
		cfg, configView,
	}

	if err := es.Publish(ctx, ev); err != nil {
		return cfg, err
	}

	return cfg, err
}

func (es *eventStore) Update(ctx context.Context, session mgauthn.Session, cfg bootstrap.Config) error {
	if err := es.svc.Update(ctx, session, cfg); err != nil {
		return err
	}

	ev := configEvent{
		cfg, configUpdate,
	}

	return es.Publish(ctx, ev)
}

func (es eventStore) UpdateCert(ctx context.Context, session mgauthn.Session, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	cfg, err := es.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
	if err != nil {
		return cfg, err
	}

	ev := updateCertEvent{
		clientID:   clientID,
		clientCert: clientCert,
		clientKey:  clientKey,
		caCert:     caCert,
	}

	if err := es.Publish(ctx, ev); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (es *eventStore) UpdateConnections(ctx context.Context, session mgauthn.Session, token, id string, connections []string) error {
	if err := es.svc.UpdateConnections(ctx, session, token, id, connections); err != nil {
		return err
	}

	ev := updateConnectionsEvent{
		mgClient:   id,
		mgChannels: connections,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) List(ctx context.Context, session mgauthn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
	bp, err := es.svc.List(ctx, session, filter, offset, limit)
	if err != nil {
		return bp, err
	}

	ev := listConfigsEvent{
		offset:       offset,
		limit:        limit,
		fullMatch:    filter.FullMatch,
		partialMatch: filter.PartialMatch,
	}

	if err := es.Publish(ctx, ev); err != nil {
		return bp, err
	}

	return bp, nil
}

func (es *eventStore) Remove(ctx context.Context, session mgauthn.Session, id string) error {
	if err := es.svc.Remove(ctx, session, id); err != nil {
		return err
	}

	ev := removeConfigEvent{
		client: id,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (bootstrap.Config, error) {
	cfg, err := es.svc.Bootstrap(ctx, externalKey, externalID, secure)

	ev := bootstrapEvent{
		cfg,
		externalID,
		true,
	}

	if err != nil {
		ev.success = false
	}

	if err := es.Publish(ctx, ev); err != nil {
		return cfg, err
	}

	return cfg, err
}

func (es *eventStore) ChangeState(ctx context.Context, session mgauthn.Session, token, id string, state bootstrap.State) error {
	if err := es.svc.ChangeState(ctx, session, token, id, state); err != nil {
		return err
	}

	ev := changeStateEvent{
		mgClient: id,
		state:    state,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) RemoveConfigHandler(ctx context.Context, id string) error {
	if err := es.svc.RemoveConfigHandler(ctx, id); err != nil {
		return err
	}

	ev := removeHandlerEvent{
		id:        id,
		operation: configHandlerRemove,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) RemoveChannelHandler(ctx context.Context, id string) error {
	if err := es.svc.RemoveChannelHandler(ctx, id); err != nil {
		return err
	}

	ev := removeHandlerEvent{
		id:        id,
		operation: channelHandlerRemove,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) error {
	if err := es.svc.UpdateChannelHandler(ctx, channel); err != nil {
		return err
	}

	ev := updateChannelHandlerEvent{
		channel,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) ConnectClientHandler(ctx context.Context, channelID, clientID string) error {
	if err := es.svc.ConnectClientHandler(ctx, channelID, clientID); err != nil {
		return err
	}

	ev := connectClientEvent{
		clientID:  clientID,
		channelID: channelID,
	}

	return es.Publish(ctx, ev)
}

func (es *eventStore) DisconnectClientHandler(ctx context.Context, channelID, clientID string) error {
	if err := es.svc.DisconnectClientHandler(ctx, channelID, clientID); err != nil {
		return err
	}

	ev := disconnectClientEvent{
		clientID:  clientID,
		channelID: channelID,
	}

	return es.Publish(ctx, ev)
}
