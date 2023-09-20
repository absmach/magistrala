// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/mainflux/mainflux/pkg/events"
	"github.com/mainflux/mainflux/pkg/events/redis"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
	"github.com/mainflux/mainflux/things/groups"
)

const streamID = "mainflux.things"

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc groups.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc groups.Service, url string) (groups.Service, error) {
	publisher, err := redis.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es *eventStore) CreateGroups(ctx context.Context, token string, groups ...mfgroups.Group) ([]mfgroups.Group, error) {
	grps, err := es.svc.CreateGroups(ctx, token, groups...)
	if err != nil {
		return grps, err
	}

	for _, group := range grps {
		event := createGroupEvent{
			group,
		}
		if err := es.Publish(ctx, event); err != nil {
			return grps, err
		}
	}
	return grps, nil
}

func (es *eventStore) UpdateGroup(ctx context.Context, token string, group mfgroups.Group) (mfgroups.Group, error) {
	group, err := es.svc.UpdateGroup(ctx, token, group)
	if err != nil {
		return group, err
	}

	event := updateGroupEvent{
		group,
	}
	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es *eventStore) ViewGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
	group, err := es.svc.ViewGroup(ctx, token, id)
	if err != nil {
		return group, err
	}
	event := viewGroupEvent{
		group,
	}
	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es *eventStore) ListGroups(ctx context.Context, token string, pm mfgroups.GroupsPage) (mfgroups.GroupsPage, error) {
	gp, err := es.svc.ListGroups(ctx, token, pm)
	if err != nil {
		return gp, err
	}
	event := listGroupEvent{
		pm,
	}
	if err := es.Publish(ctx, event); err != nil {
		return gp, err
	}

	return gp, nil
}

func (es *eventStore) ListMemberships(ctx context.Context, token, clientID string, pm mfgroups.GroupsPage) (mfgroups.MembershipsPage, error) {
	mp, err := es.svc.ListMemberships(ctx, token, clientID, pm)
	if err != nil {
		return mp, err
	}
	event := listGroupMembershipEvent{
		pm, clientID,
	}
	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es *eventStore) EnableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
	group, err := es.svc.EnableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es *eventStore) DisableGroup(ctx context.Context, token, id string) (mfgroups.Group, error) {
	group, err := es.svc.DisableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es *eventStore) delete(ctx context.Context, group mfgroups.Group) (mfgroups.Group, error) {
	event := removeGroupEvent{
		id:        group.ID,
		updatedAt: group.UpdatedAt,
		updatedBy: group.UpdatedBy,
		status:    group.Status.String(),
	}
	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}
