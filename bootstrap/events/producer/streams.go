// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer

import (
	"context"

	"github.com/absmach/magistrala/bootstrap"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
)

var _ bootstrap.Service = (*eventStore)(nil)

const (
	magistralaPrefix    = "magistrala."
	createStream        = magistralaPrefix + configCreate
	viewStream          = magistralaPrefix + configView
	listStream          = magistralaPrefix + configList
	updateStream        = magistralaPrefix + configUpdate
	removeStream        = magistralaPrefix + configRemove
	updateCertStream    = magistralaPrefix + certUpdate
	removeHandlerStream = magistralaPrefix + configHandlerRemove
	bootstrapStream     = magistralaPrefix + clientBootstrap
	enableConfigStream  = magistralaPrefix + clientEnable
	disableConfigStream = magistralaPrefix + clientDisable
)

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

func (es *eventStore) Add(ctx context.Context, session smqauthn.Session, token string, cfg bootstrap.Config) (bootstrap.Config, error) {
	saved, err := es.svc.Add(ctx, session, token, cfg)
	if err != nil {
		return saved, err
	}

	ev := configEvent{
		saved, configCreate,
	}

	if err := es.Publish(ctx, createStream, ev); err != nil {
		return saved, err
	}

	return saved, err
}

func (es *eventStore) View(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	cfg, err := es.svc.View(ctx, session, id)
	if err != nil {
		return cfg, err
	}
	ev := configEvent{
		cfg, configView,
	}

	if err := es.Publish(ctx, configView, ev); err != nil {
		return cfg, err
	}

	return cfg, err
}

func (es *eventStore) Update(ctx context.Context, session smqauthn.Session, cfg bootstrap.Config) error {
	if err := es.svc.Update(ctx, session, cfg); err != nil {
		return err
	}

	ev := configEvent{
		cfg, configUpdate,
	}

	return es.Publish(ctx, configUpdate, ev)
}

func (es eventStore) UpdateCert(ctx context.Context, session smqauthn.Session, clientID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	cfg, err := es.svc.UpdateCert(ctx, session, clientID, clientCert, clientKey, caCert)
	if err != nil {
		return cfg, err
	}

	ev := updateCertEvent{
		configID:   clientID,
		clientCert: clientCert,
		clientKey:  clientKey,
		caCert:     caCert,
	}

	if err := es.Publish(ctx, updateCertStream, ev); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (es *eventStore) List(ctx context.Context, session smqauthn.Session, filter bootstrap.Filter, offset, limit uint64) (bootstrap.ConfigsPage, error) {
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

	if err := es.Publish(ctx, listStream, ev); err != nil {
		return bp, err
	}

	return bp, nil
}

func (es *eventStore) Remove(ctx context.Context, session smqauthn.Session, id string) error {
	if err := es.svc.Remove(ctx, session, id); err != nil {
		return err
	}

	ev := removeConfigEvent{
		config: id,
	}

	return es.Publish(ctx, removeStream, ev)
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

	if err := es.Publish(ctx, bootstrapStream, ev); err != nil {
		return cfg, err
	}

	return cfg, err
}

func (es *eventStore) EnableConfig(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	cfg, err := es.svc.EnableConfig(ctx, session, id)
	if err != nil {
		return cfg, err
	}

	ev := enableConfigEvent{configID: id}
	if err := es.Publish(ctx, enableConfigStream, ev); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (es *eventStore) DisableConfig(ctx context.Context, session smqauthn.Session, id string) (bootstrap.Config, error) {
	cfg, err := es.svc.DisableConfig(ctx, session, id)
	if err != nil {
		return cfg, err
	}

	ev := disableConfigEvent{configID: id}
	if err := es.Publish(ctx, disableConfigStream, ev); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (es *eventStore) RemoveConfigHandler(ctx context.Context, id string) error {
	if err := es.svc.RemoveConfigHandler(ctx, id); err != nil {
		return err
	}

	ev := removeHandlerEvent{
		id:        id,
		operation: configHandlerRemove,
	}

	return es.Publish(ctx, removeHandlerStream, ev)
}

func (es *eventStore) CreateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) (bootstrap.Profile, error) {
	return es.svc.CreateProfile(ctx, session, p)
}

func (es *eventStore) ViewProfile(ctx context.Context, session smqauthn.Session, profileID string) (bootstrap.Profile, error) {
	return es.svc.ViewProfile(ctx, session, profileID)
}

func (es *eventStore) UpdateProfile(ctx context.Context, session smqauthn.Session, p bootstrap.Profile) error {
	return es.svc.UpdateProfile(ctx, session, p)
}

func (es *eventStore) ListProfiles(ctx context.Context, session smqauthn.Session, offset, limit uint64) (bootstrap.ProfilesPage, error) {
	return es.svc.ListProfiles(ctx, session, offset, limit)
}

func (es *eventStore) DeleteProfile(ctx context.Context, session smqauthn.Session, profileID string) error {
	return es.svc.DeleteProfile(ctx, session, profileID)
}

func (es *eventStore) AssignProfile(ctx context.Context, session smqauthn.Session, configID, profileID string) error {
	return es.svc.AssignProfile(ctx, session, configID, profileID)
}

func (es *eventStore) BindResources(ctx context.Context, session smqauthn.Session, token, configID string, bindings []bootstrap.BindingRequest) error {
	return es.svc.BindResources(ctx, session, token, configID, bindings)
}

func (es *eventStore) ListBindings(ctx context.Context, session smqauthn.Session, configID string) ([]bootstrap.BindingSnapshot, error) {
	return es.svc.ListBindings(ctx, session, configID)
}

func (es *eventStore) RefreshBindings(ctx context.Context, session smqauthn.Session, token, configID string) error {
	return es.svc.RefreshBindings(ctx, session, token, configID)
}
