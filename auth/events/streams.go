// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	"github.com/absmach/magistrala/pkg/policy"
)

const streamID = "magistrala.auth"

var _ auth.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc auth.Service
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc auth.Service, url string) (auth.Service, error) {
	publisher, err := store.NewPublisher(ctx, url, streamID)
	if err != nil {
		return nil, err
	}

	return &eventStore{
		svc:       svc,
		Publisher: publisher,
	}, nil
}

func (es *eventStore) CreateDomain(ctx context.Context, token string, domain auth.Domain) (auth.Domain, error) {
	domain, err := es.svc.CreateDomain(ctx, token, domain)
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

func (es *eventStore) RetrieveDomain(ctx context.Context, token, id string) (auth.Domain, error) {
	domain, err := es.svc.RetrieveDomain(ctx, token, id)
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

func (es *eventStore) RetrieveDomainPermissions(ctx context.Context, token, id string) (policy.Permissions, error) {
	permissions, err := es.svc.RetrieveDomainPermissions(ctx, token, id)
	if err != nil {
		return permissions, err
	}

	event := retrieveDomainPermissionsEvent{
		domainID:    id,
		permissions: permissions,
	}

	if err := es.Publish(ctx, event); err != nil {
		return permissions, err
	}

	return permissions, nil
}

func (es *eventStore) UpdateDomain(ctx context.Context, token, id string, d auth.DomainReq) (auth.Domain, error) {
	domain, err := es.svc.UpdateDomain(ctx, token, id, d)
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

func (es *eventStore) ChangeDomainStatus(ctx context.Context, token, id string, d auth.DomainReq) (auth.Domain, error) {
	domain, err := es.svc.ChangeDomainStatus(ctx, token, id, d)
	if err != nil {
		return domain, err
	}

	event := changeDomainStatusEvent{
		domainID:  id,
		status:    domain.Status,
		updatedAt: domain.UpdatedAt,
		updatedBy: domain.UpdatedBy,
	}

	if err := es.Publish(ctx, event); err != nil {
		return domain, err
	}

	return domain, nil
}

func (es *eventStore) ListDomains(ctx context.Context, token string, p auth.Page) (auth.DomainsPage, error) {
	dp, err := es.svc.ListDomains(ctx, token, p)
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

func (es *eventStore) AssignUsers(ctx context.Context, token, id string, userIds []string, relation string) error {
	err := es.svc.AssignUsers(ctx, token, id, userIds, relation)
	if err != nil {
		return err
	}

	event := assignUsersEvent{
		domainID: id,
		userIDs:  userIds,
		relation: relation,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) UnassignUser(ctx context.Context, token, id, userID string) error {
	err := es.svc.UnassignUser(ctx, token, id, userID)
	if err != nil {
		return err
	}

	event := unassignUsersEvent{
		domainID: id,
		userID:   userID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return err
	}

	return nil
}

func (es *eventStore) ListUserDomains(ctx context.Context, token, userID string, p auth.Page) (auth.DomainsPage, error) {
	dp, err := es.svc.ListUserDomains(ctx, token, userID, p)
	if err != nil {
		return dp, err
	}

	event := listUserDomainsEvent{
		Page:   p,
		userID: userID,
	}

	if err := es.Publish(ctx, event); err != nil {
		return dp, err
	}

	return dp, nil
}

func (es *eventStore) Issue(ctx context.Context, token string, key auth.Key) (auth.Token, error) {
	return es.svc.Issue(ctx, token, key)
}

func (es *eventStore) Revoke(ctx context.Context, token, id string) error {
	return es.svc.Revoke(ctx, token, id)
}

func (es *eventStore) RetrieveKey(ctx context.Context, token, id string) (auth.Key, error) {
	return es.svc.RetrieveKey(ctx, token, id)
}

func (es *eventStore) Identify(ctx context.Context, token string) (auth.Key, error) {
	return es.svc.Identify(ctx, token)
}

func (es *eventStore) Authorize(ctx context.Context, pr auth.PolicyReq) error {
	return es.svc.Authorize(ctx, pr)
}

func (es *eventStore) DeleteUserPolicies(ctx context.Context, id string) error {
	return es.svc.DeleteUserPolicies(ctx, id)
}
