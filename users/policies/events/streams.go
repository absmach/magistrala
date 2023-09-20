// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/redis"
	"github.com/mainflux/mainflux/users/policies"
)

const streamID = "mainflux.users"

var _ policies.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc policies.Service
}

// NewEventStoreMiddleware returns wrapper around policy service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc policies.Service, url string) (policies.Service, error) {
	publisher, err := redis.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es *eventStore) Authorize(ctx context.Context, ar policies.AccessRequest) error {
	if err := es.svc.Authorize(ctx, ar); err != nil {
		return err
	}

	event := authorizeEvent{
		ar,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) AddPolicy(ctx context.Context, token string, policy policies.Policy) error {
	if err := es.svc.AddPolicy(ctx, token, policy); err != nil {
		return err
	}

	event := policyEvent{
		policy, policyAdd,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) UpdatePolicy(ctx context.Context, token string, policy policies.Policy) error {
	if err := es.svc.UpdatePolicy(ctx, token, policy); err != nil {
		return err
	}

	event := policyEvent{
		policy, policyUpdate,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) ListPolicies(ctx context.Context, token string, page policies.Page) (policies.PolicyPage, error) {
	pp, err := es.svc.ListPolicies(ctx, token, page)
	if err != nil {
		return pp, err
	}

	event := listPoliciesEvent{
		page,
	}

	if err := es.Publish(ctx, event); err != nil {
		return pp, err
	}

	return pp, nil
}

func (es *eventStore) DeletePolicy(ctx context.Context, token string, policy policies.Policy) error {
	if err := es.svc.DeletePolicy(ctx, token, policy); err != nil {
		return err
	}

	event := policyEvent{
		policy, policyDelete,
	}

	return es.Publish(ctx, event)
}
