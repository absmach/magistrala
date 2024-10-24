// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rmEvents "github.com/absmach/magistrala/pkg/roles/rolemanager/events"
)

const streamID = "magistrala.domains"

var _ domains.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc domains.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc domains.Service, url string) (domains.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	res := rmEvents.NewRoleManagerEventStore("domains", svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) CreateDomain(ctx context.Context, session authn.Session, domain domains.Domain) (domains.Domain, error) {
	domain, err := es.svc.CreateDomain(ctx, session, domain)
	if err != nil {
		return domain, err
	}

	event := createDomainEvent{
		domain,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) RetrieveDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	domain, err := es.svc.RetrieveDomain(ctx, session, id)
	if err != nil {
		return domain, err
	}

	event := retrieveDomainEvent{
		domain,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) UpdateDomain(ctx context.Context, session authn.Session, id string, d domains.DomainReq) (domains.Domain, error) {
	domain, err := es.svc.UpdateDomain(ctx, session, id, d)
	if err != nil {
		return domain, err
	}

	event := updateDomainEvent{
		domain,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) EnableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	domain, err := es.svc.EnableDomain(ctx, session, id)
	if err != nil {
		return domain, err
	}

	event := enableDomainEvent{
		domainID:  id,
		updatedAt: domain.UpdatedAt,
		updatedBy: domain.UpdatedBy,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) DisableDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	domain, err := es.svc.DisableDomain(ctx, session, id)
	if err != nil {
		return domain, err
	}

	event := disableDomainEvent{
		domainID:  id,
		updatedAt: domain.UpdatedAt,
		updatedBy: domain.UpdatedBy,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) FreezeDomain(ctx context.Context, session authn.Session, id string) (domains.Domain, error) {
	domain, err := es.svc.FreezeDomain(ctx, session, id)
	if err != nil {
		return domain, err
	}

	event := freezeDomainEvent{
		domainID:  id,
		updatedAt: domain.UpdatedAt,
		updatedBy: domain.UpdatedBy,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) ListDomains(ctx context.Context, session authn.Session, p domains.Page) (domains.DomainsPage, error) {
	dp, err := es.svc.ListDomains(ctx, session, p)
	if err != nil {
		return dp, err
	}

	event := listDomainsEvent{
		p, dp.Total,
	}

	if err := es.Publish(ctx, event); err != nil {
		return dp, err
	}

	return dp, nil
}

func (es *eventStore) DeleteUserFromDomains(ctx context.Context, userID string) error {
	if err := es.svc.DeleteUserFromDomains(ctx, userID); err != nil {
		return err
	}

	event := deleteUserFromDomainsEvent{userID}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}
