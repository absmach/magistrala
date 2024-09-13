// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/pkg/auth"
	mgclients "github.com/absmach/magistrala/pkg/clients"
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

func (es *eventStore) CreateThings(ctx context.Context, session auth.Session, thing ...mgclients.Client) ([]mgclients.Client, error) {
	sths, err := es.svc.CreateThings(ctx, session, thing...)
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

func (es *eventStore) UpdateClient(ctx context.Context, session auth.Session, thing mgclients.Client) (mgclients.Client, error) {
	cli, err := es.svc.UpdateClient(ctx, session, thing)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "", cli)
}

func (es *eventStore) UpdateClientTags(ctx context.Context, session auth.Session, thing mgclients.Client) (mgclients.Client, error) {
	cli, err := es.svc.UpdateClientTags(ctx, session, thing)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "tags", cli)
}

func (es *eventStore) UpdateClientSecret(ctx context.Context, session auth.Session, id, key string) (mgclients.Client, error) {
	cli, err := es.svc.UpdateClientSecret(ctx, session, id, key)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "secret", cli)
}

func (es *eventStore) update(ctx context.Context, operation string, thing mgclients.Client) (mgclients.Client, error) {
	event := updateClientEvent{
		thing, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return thing, err
	}

	return thing, nil
}

func (es *eventStore) ViewClient(ctx context.Context, id string) (mgclients.Client, error) {
	cli, err := es.svc.ViewClient(ctx, id)
	if err != nil {
		return cli, err
	}

	event := viewClientEvent{
		cli,
	}
	if err := es.Publish(ctx, event); err != nil {
		return cli, err
	}

	return cli, nil
}

func (es *eventStore) ViewClientPerms(ctx context.Context, session auth.Session, id string) ([]string, error) {
	permissions, err := es.svc.ViewClientPerms(ctx, session, id)
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

func (es *eventStore) ListClients(ctx context.Context, session auth.Session, reqUserID string, pm mgclients.Page) (mgclients.ClientsPage, error) {
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

func (es *eventStore) ListClientsByGroup(ctx context.Context, session auth.Session, chID string, pm mgclients.Page) (mgclients.MembersPage, error) {
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

func (es *eventStore) EnableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	cli, err := es.svc.EnableClient(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) DisableClient(ctx context.Context, session auth.Session, id string) (mgclients.Client, error) {
	cli, err := es.svc.DisableClient(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) changeStatus(ctx context.Context, cli mgclients.Client) (mgclients.Client, error) {
	event := changeStatusClientEvent{
		id:        cli.ID,
		updatedAt: cli.UpdatedAt,
		updatedBy: cli.UpdatedBy,
		status:    cli.Status.String(),
	}
	if err := es.Publish(ctx, event); err != nil {
		return cli, err
	}

	return cli, nil
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

func (es *eventStore) Share(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
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

func (es *eventStore) Unshare(ctx context.Context, session auth.Session, id, relation string, userids ...string) error {
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

func (es *eventStore) DeleteClient(ctx context.Context, id string) error {
	if err := es.svc.DeleteClient(ctx, id); err != nil {
		return err
	}

	event := removeClientEvent{id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
