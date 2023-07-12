// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/twins"
)

const (
	streamID  = "mainflux.twins"
	streamLen = 1000
)

var _ twins.Service = (*eventStore)(nil)

type eventStore struct {
	svc    twins.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(svc twins.Service, client *redis.Client) twins.Service {
	return eventStore{
		svc:    svc,
		client: client,
	}
}

func (es eventStore) AddTwin(ctx context.Context, token string, twin twins.Twin, def twins.Definition) (twins.Twin, error) {
	twin, err := es.svc.AddTwin(ctx, token, twin, def)
	if err != nil {
		return twin, err
	}

	event := addTwinEvent{
		twin, def,
	}
	values, err := event.Encode()
	if err != nil {
		return twin, err
	}
	record := &redis.XAddArgs{
		Stream: streamID,
		MaxLen: streamLen,
		Values: values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
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
	values, err := event.Encode()
	if err != nil {
		return err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
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
	values, err := event.Encode()
	if err != nil {
		return twin, err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
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
	values, err := event.Encode()
	if err != nil {
		return err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
		return err
	}

	return nil
}

func (es eventStore) ListTwins(ctx context.Context, token string, offset uint64, limit uint64, name string, metadata twins.Metadata) (twins.Page, error) {
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
	values, err := event.Encode()
	if err != nil {
		return tp, err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
		return tp, err
	}

	return tp, nil
}

func (es eventStore) ListStates(ctx context.Context, token string, offset uint64, limit uint64, id string) (twins.StatesPage, error) {
	sp, err := es.svc.ListStates(ctx, token, offset, limit, id)
	if err != nil {
		return sp, err
	}
	event := listStatesEvent{
		offset,
		limit,
		id,
	}
	values, err := event.Encode()
	if err != nil {
		return sp, err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
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
	values, err := event.Encode()
	if err != nil {
		return err
	}
	record := &redis.XAddArgs{
		Stream:       streamID,
		MaxLenApprox: streamLen,
		Values:       values,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
		return err
	}

	return nil
}
