// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/groups"
)

const streamID = "magistrala.users"

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc groups.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc groups.Service, url string) (groups.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es eventStore) CreateGroup(ctx context.Context, token, kind string, group groups.Group) (groups.Group, error) {
	group, err := es.svc.CreateGroup(ctx, token, kind, group)
	if err != nil {
		return group, err
	}

	event := createGroupEvent{
		group,
	}

	if err := es.Publish(ctx, event); err != nil {
		return group, err
	}

	return group, nil
}

func (es eventStore) UpdateGroup(ctx context.Context, token string, group groups.Group) (groups.Group, error) {
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

func (es eventStore) ViewGroup(ctx context.Context, token, id string) (groups.Group, error) {
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

func (es eventStore) ListGroups(ctx context.Context, token, memberKind, memberID string, pm groups.Page) (groups.Page, error) {
	gp, err := es.svc.ListGroups(ctx, token, memberKind, memberID, pm)
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

func (es eventStore) ListMembers(ctx context.Context, token, groupID, permission, memberKind string) (groups.MembersPage, error) {
	mp, err := es.svc.ListMembers(ctx, token, groupID, permission, memberKind)
	if err != nil {
		return mp, err
	}
	event := listGroupMembershipEvent{
		groupID, permission, memberKind,
	}

	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es eventStore) EnableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group, err := es.svc.EnableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es eventStore) Assign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	return es.svc.Assign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) Unassign(ctx context.Context, token, groupID, relation, memberKind string, memberIDs ...string) error {
	return es.svc.Unassign(ctx, token, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) DisableGroup(ctx context.Context, token, id string) (groups.Group, error) {
	group, err := es.svc.DisableGroup(ctx, token, id)
	if err != nil {
		return group, err
	}

	return es.delete(ctx, group)
}

func (es eventStore) delete(ctx context.Context, group groups.Group) (groups.Group, error) {
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
