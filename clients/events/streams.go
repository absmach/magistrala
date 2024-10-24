// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rmEvents "github.com/absmach/magistrala/pkg/roles/rolemanager/events"
)

const streamID = "magistrala.clients"

var _ clients.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc clients.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around clients service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc clients.Service, url string) (clients.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}
	res := rmEvents.NewRoleManagerEventStore("clients", svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) CreateClients(ctx context.Context, session authn.Session, clients ...clients.Client) ([]clients.Client, error) {
	clis, err := es.svc.CreateClients(ctx, session, clients...)
	if err != nil {
		return clis, err
	}

	for _, cli := range clis {
		event := createClientEvent{
			cli,
		}
		if err := es.Publish(ctx, event); err != nil {
			return clis, err
		}
	}

	return clis, nil
}

func (es *eventStore) Update(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	cli, err := es.svc.Update(ctx, session, client)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "", cli)
}

func (es *eventStore) UpdateTags(ctx context.Context, session authn.Session, client clients.Client) (clients.Client, error) {
	cli, err := es.svc.UpdateTags(ctx, session, client)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "tags", cli)
}

func (es *eventStore) UpdateSecret(ctx context.Context, session authn.Session, id, key string) (clients.Client, error) {
	cli, err := es.svc.UpdateSecret(ctx, session, id, key)
	if err != nil {
		return cli, err
	}

	return es.update(ctx, "secret", cli)
}

func (es *eventStore) update(ctx context.Context, operation string, client clients.Client) (clients.Client, error) {
	event := updateClientEvent{
		client, operation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return client, err
	}

	return client, nil
}

func (es *eventStore) View(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	cli, err := es.svc.View(ctx, session, id)
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

func (es *eventStore) ListClients(ctx context.Context, session authn.Session, reqUserID string, pm clients.Page) (clients.ClientsPage, error) {
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

func (es *eventStore) Enable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	cli, err := es.svc.Enable(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) Disable(ctx context.Context, session authn.Session, id string) (clients.Client, error) {
	cli, err := es.svc.Disable(ctx, session, id)
	if err != nil {
		return cli, err
	}

	return es.changeStatus(ctx, cli)
}

func (es *eventStore) changeStatus(ctx context.Context, cli clients.Client) (clients.Client, error) {
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

func (es *eventStore) SetParentGroup(ctx context.Context, session authn.Session, parentGroupID string, id string) (err error) {
	if err := es.svc.SetParentGroup(ctx, session, parentGroupID, id); err != nil {
		return err
	}

	event := setParentGroupEvent{parentGroupID: parentGroupID, id: id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) RemoveParentGroup(ctx context.Context, session authn.Session, id string) (err error) {
	if err := es.svc.RemoveParentGroup(ctx, session, id); err != nil {
		return err
	}

	event := removeParentGroupEvent{id: id}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
