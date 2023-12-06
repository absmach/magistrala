// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/twins"
)

const streamID = "magistrala.twins"

var _ twins.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc twins.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc twins.Service, url string) (twins.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es eventStore) AddTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (twins.Twin, error) {
	twin, err := es.svc.AddTwin(ctx, token, twin, def)
	if err != nil {
		return twin, err
	}

	event := addTwinEvent{
		twin, def,
	}

	if err := es.Publish(ctx, event); err != nil {
		return twin, err
	}

	return twin, nil
}

func (es eventStore) UpdateTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) error {
	if err := es.svc.UpdateTwin(ctx, token, twin, def); err != nil {
		return err
	}

	event := updateTwinEvent{
		twin, def,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es eventStore) ViewTwin(ctx context.Context, token, id string) (twins.Twin, error) {
	twin, err := es.svc.ViewTwin(ctx, token, id)
	if err != nil {
		return twin, err
	}

	event := viewTwinEvent{
		id,
	}

	if err := es.Publish(ctx, event); err != nil {
		return twin, err
	}

	return twin, nil
}

func (es eventStore) RemoveTwin(ctx context.Context, token, id string) error {
	if err := es.svc.RemoveTwin(ctx, token, id); err != nil {
		return err
	}

	event := removeTwinEvent{
		id,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es eventStore) ListTwins(ctx context.Context, token string, offset, limit uint64, name string, metadata twins.Metadata) (twins.Page, error) {
	tp, err := es.svc.ListTwins(ctx, token, offset, limit, name, metadata)
	if err != nil {
		return tp, err
	}
	event := listTwinsEvent{
		offset,
		limit,
		name,
		metadata,
	}

	if err := es.Publish(ctx, event); err != nil {
		return tp, err
	}

	return tp, nil
}

func (es eventStore) ListStates(ctx context.Context, token string, offset, limit uint64, id string) (twins.StatesPage, error) {
	sp, err := es.svc.ListStates(ctx, token, offset, limit, id)
	if err != nil {
		return sp, err
	}

	event := listStatesEvent{
		offset,
		limit,
		id,
	}

	if err := es.Publish(ctx, event); err != nil {
		return sp, err
	}

	return sp, nil
}

func (es eventStore) SaveStates(ctx context.Context, msg *messaging.Message) error {
	if err := es.svc.SaveStates(ctx, msg); err != nil {
		return err
	}
	event := saveStatesEvent{
		msg,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
