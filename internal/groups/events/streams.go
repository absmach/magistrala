// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/groups"
)

var _ groups.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc groups.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc groups.Service, url, streamID string) (groups.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es eventStore) CreateGroup(ctx context.Context, session authn.Session, kind string, group groups.Group) (groups.Group, error) {
	group, err := es.svc.CreateGroup(ctx, session, kind, group)
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

func (es eventStore) UpdateGroup(ctx context.Context, session authn.Session, group groups.Group) (groups.Group, error) {
	group, err := es.svc.UpdateGroup(ctx, session, group)
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

func (es eventStore) ViewGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	group, err := es.svc.ViewGroup(ctx, session, id)
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

func (es eventStore) ViewGroupPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	permissions, err := es.svc.ViewGroupPerms(ctx, session, id)
	if err != nil {
		return permissions, err
	}
	event := viewGroupPermsEvent{
		permissions,
	}

	if err := es.Publish(ctx, event); err != nil {
		return permissions, err
	}

	return permissions, nil
}

func (es eventStore) ListGroups(ctx context.Context, session authn.Session, memberKind, memberID string, pm groups.Page) (groups.Page, error) {
	gp, err := es.svc.ListGroups(ctx, session, memberKind, memberID, pm)
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

func (es eventStore) ListMembers(ctx context.Context, session authn.Session, groupID, permission, memberKind string) (groups.MembersPage, error) {
	mp, err := es.svc.ListMembers(ctx, session, groupID, permission, memberKind)
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

func (es eventStore) EnableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	group, err := es.svc.EnableGroup(ctx, session, id)
	if err != nil {
		return group, err
	}

	return es.changeStatus(ctx, group)
}

func (es eventStore) Assign(ctx context.Context, session authn.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	if err := es.svc.Assign(ctx, session, groupID, relation, memberKind, memberIDs...); err != nil {
		return err
	}

	event := assignEvent{
		groupID:    groupID,
		relation:   relation,
		memberKind: memberKind,
		memberIDs:  memberIDs,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es eventStore) Unassign(ctx context.Context, session authn.Session, groupID, relation, memberKind string, memberIDs ...string) error {
	if err := es.svc.Unassign(ctx, session, groupID, relation, memberKind, memberIDs...); err != nil {
		return err
	}

	event := unassignEvent{
		groupID:    groupID,
		relation:   relation,
		memberKind: memberKind,
		memberIDs:  memberIDs,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}
	return es.svc.Unassign(ctx, session, groupID, relation, memberKind, memberIDs...)
}

func (es eventStore) DisableGroup(ctx context.Context, session authn.Session, id string) (groups.Group, error) {
	group, err := es.svc.DisableGroup(ctx, session, id)
	if err != nil {
		return group, err
	}

	return es.changeStatus(ctx, group)
}

func (es eventStore) changeStatus(ctx context.Context, group groups.Group) (groups.Group, error) {
	event := changeStatusGroupEvent{
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

func (es eventStore) DeleteGroup(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.DeleteGroup(ctx, session, id); err != nil {
		return err
	}
	if err := es.Publish(ctx, deleteGroupEvent{id}); err != nil {
		return err
	}
	return nil
}
