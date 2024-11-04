// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/things"
)

const streamID = "magistrala.things"

var _ things.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc things.Service
}

// NewEventStoreMiddleware returns wrapper around things service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc things.Service, url string) (things.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es *eventStore) CreateClients(ctx context.Context, session authn.Session, thing ...things.Client) ([]things.Client, error) {
	sths, err := es.svc.CreateClients(ctx, session, thing...)
	if err != nil {
		return sths, err
	}

	for _, th := range sths {
		event := createClientEvent{
			th,
		}
		if err := es.Publish(ctx, event); err != nil {
			return sths, err
		}
	}

	return sths, nil
}

func (es *eventStore) Update(ctx context.Context, session authn.Session, thing things.Client) (things.Client, error) {
	cli, err := es.svc.Update(ctx, session, thing)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "", cli)
}

func (es *eventStore) UpdateTags(ctx context.Context, session authn.Session, thing things.Client) (things.Client, error) {
	cli, err := es.svc.UpdateTags(ctx, session, thing)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "tags", cli)
}

func (es *eventStore) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (things.Client, error) {
	cli, err := es.svc.UpdateSecret(ctx, session, id, key)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "secret", cli)
}

func (es *eventStore) update(ctx context.Context, operation string, thing things.Client) (things.Client, error) {
	event := updateClientEvent{
		thing, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return thing, err
	}

	return thing, nil
}

func (es *eventStore) View(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	thi, err := es.svc.View(ctx, session, id)
	if err != nil {
		return thi, err
	}

	event := viewClientEvent{
		thi,
	}
	if err := es.Publish(ctx, event); err != nil {
		return thi, err
	}

	return thi, nil
}

func (es *eventStore) ViewPerms(ctx context.Context, session authn.Session, id string) ([]string, error) {
	permissions, err := es.svc.ViewPerms(ctx, session, id)
	if err != nil {
		return permissions, err
	}

	event := viewClientPermsEvent{
		permissions,
	}
	if err := es.Publish(ctx, event); err != nil {
		return permissions, err
	}

	return permissions, nil
}

func (es *eventStore) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm things.Page) (things.ClientsPage, error) {
	cp, err := es.svc.ListClients(ctx, session, reqUserID, pm)
	if err != nil {
		return cp, err
	}
	event := listClientEvent{
		reqUserID,
		pm,
	}
	if err := es.Publish(ctx, event); err != nil {
		return cp, err
	}

	return cp, nil
}

func (es *eventStore) ListClientsByGroup(ctx context.Context, session authn.Session, chID string, pm things.Page) (things.MembersPage, error) {
	mp, err := es.svc.ListClientsByGroup(ctx, session, chID, pm)
	if err != nil {
		return mp, err
	}
	event := listClientByGroupEvent{
		pm, chID,
	}
	if err := es.Publish(ctx, event); err != nil {
		return mp, err
	}

	return mp, nil
}

func (es *eventStore) Enable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	thi, err := es.svc.Enable(ctx, session, id)
	if err != nil {
		return thi, err
	}

	return es.changeStatus(ctx, thi)
}

func (es *eventStore) Disable(ctx context.Context, session authn.Session, id string) (things.Client, error) {
	thi, err := es.svc.Disable(ctx, session, id)
	if err != nil {
		return thi, err
	}

	return es.changeStatus(ctx, thi)
}

func (es *eventStore) changeStatus(ctx context.Context, thi things.Client) (things.Client, error) {
	event := changeStatusClientEvent{
		id:        thi.ID,
		updatedAt: thi.UpdatedAt,
		updatedBy: thi.UpdatedBy,
		status:    thi.Status.String(),
	}
	if err := es.Publish(ctx, event); err != nil {
		return thi, err
	}

	return thi, nil
}

func (es *eventStore) Identify(ctx context.Context, key string) (string, error) {
	thingID, err := es.svc.Identify(ctx, key)
	if err != nil {
		return thingID, err
	}
	event := identifyClientEvent{
		thingID: thingID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return thingID, err
	}
	return thingID, nil
}

func (es *eventStore) Authorize(ctx context.Context, req things.AuthzReq) (string, error) {
	thingID, err := es.svc.Authorize(ctx, req)
	if err != nil {
		return thingID, err
	}

	event := authorizeClientEvent{
		thingID:    thingID,
		channelID:  req.ChannelID,
		permission: req.Permission,
	}

	if err := es.Publish(ctx, event); err != nil {
		return thingID, err
	}

	return thingID, nil
}

func (es *eventStore) Share(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	if err := es.svc.Share(ctx, session, id, relation, userids...); err != nil {
		return err
	}

	event := shareClientEvent{
		action:   "share",
		id:       id,
		relation: relation,
		userIDs:  userids,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) Unshare(ctx context.Context, session authn.Session, id, relation string, userids ...string) error {
	if err := es.svc.Unshare(ctx, session, id, relation, userids...); err != nil {
		return err
	}

	event := shareClientEvent{
		action:   "unshare",
		id:       id,
		relation: relation,
		userIDs:  userids,
	}

	return es.Publish(ctx, event)
}

func (es *eventStore) Delete(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.Delete(ctx, session, id); err != nil {
		return err
	}

	event := removeClientEvent{id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
