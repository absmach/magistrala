// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
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

func (es *eventStore) RetrieveDomainPermissions(ctx context.Context, token, id string) (auth.Permissions, error) {
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

func (es *eventStore) AddPolicy(ctx context.Context, pr auth.PolicyReq) error {
	return es.svc.AddPolicy(ctx, pr)
}

func (es *eventStore) AddPolicies(ctx context.Context, prs []auth.PolicyReq) error {
	return es.svc.AddPolicies(ctx, prs)
}

func (es *eventStore) DeletePolicyFilter(ctx context.Context, pr auth.PolicyReq) error {
	return es.svc.DeletePolicyFilter(ctx, pr)
}

func (es *eventStore) DeleteEntityPolicies(ctx context.Context, entityType, id string) error {
	return es.svc.DeleteEntityPolicies(ctx, entityType, id)
}

func (es *eventStore) DeletePolicies(ctx context.Context, prs []auth.PolicyReq) error {
	return es.svc.DeletePolicies(ctx, prs)
}

func (es *eventStore) ListObjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit uint64) (auth.PolicyPage, error) {
	return es.svc.ListObjects(ctx, pr, nextPageToken, limit)
}

func (es *eventStore) ListAllObjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	return es.svc.ListAllObjects(ctx, pr)
}

func (es *eventStore) CountObjects(ctx context.Context, pr auth.PolicyReq) (uint64, error) {
	return es.svc.CountObjects(ctx, pr)
}

func (es *eventStore) ListSubjects(ctx context.Context, pr auth.PolicyReq, nextPageToken string, limit uint64) (auth.PolicyPage, error) {
	return es.svc.ListSubjects(ctx, pr, nextPageToken, limit)
}

func (es *eventStore) ListAllSubjects(ctx context.Context, pr auth.PolicyReq) (auth.PolicyPage, error) {
	return es.svc.ListAllSubjects(ctx, pr)
}

func (es *eventStore) CountSubjects(ctx context.Context, pr auth.PolicyReq) (uint64, error) {
	return es.svc.CountSubjects(ctx, pr)
}

func (es *eventStore) ListPermissions(ctx context.Context, pr auth.PolicyReq, filterPermission []string) (auth.Permissions, error) {
	return es.svc.ListPermissions(ctx, pr, filterPermission)
}

func (es *eventStore) CreatePAT(ctx context.Context, token, name, description string, duration time.Duration, scope auth.Scope) (auth.PAT, error) {
	return es.svc.CreatePAT(ctx, token, name, description, duration, scope)
}

func (es *eventStore) UpdatePATName(ctx context.Context, token, patID, name string) (auth.PAT, error) {
	return es.svc.UpdatePATName(ctx, token, patID, name)
}

func (es *eventStore) UpdatePATDescription(ctx context.Context, token, patID, description string) (auth.PAT, error) {
	return es.svc.UpdatePATDescription(ctx, token, patID, description)
}

func (es *eventStore) RetrievePAT(ctx context.Context, token, patID string) (auth.PAT, error) {
	return es.svc.RetrievePAT(ctx, token, patID)
}

func (es *eventStore) ListPATS(ctx context.Context, token string, pm auth.PATSPageMeta) (auth.PATSPage, error) {
	return es.svc.ListPATS(ctx, token, pm)
}

func (es *eventStore) DeletePAT(ctx context.Context, token, patID string) error {
	return es.svc.DeletePAT(ctx, token, patID)
}

func (es *eventStore) ResetPATSecret(ctx context.Context, token, patID string, duration time.Duration) (auth.PAT, error) {
	return es.svc.ResetPATSecret(ctx, token, patID, duration)
}

func (es *eventStore) RevokePATSecret(ctx context.Context, token, patID string) error {
	return es.svc.RevokePATSecret(ctx, token, patID)
}

func (es *eventStore) AddPATScopeEntry(ctx context.Context, token, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	return es.svc.AddPATScopeEntry(ctx, token, patID, platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
}

func (es *eventStore) RemovePATScopeEntry(ctx context.Context, token, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) (auth.Scope, error) {
	return es.svc.RemovePATScopeEntry(ctx, token, patID, platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
}

func (es *eventStore) ClearPATAllScopeEntry(ctx context.Context, token, patID string) error {
	return es.svc.ClearPATAllScopeEntry(ctx, token, patID)
}

func (es *eventStore) IdentifyPAT(ctx context.Context, paToken string) (auth.PAT, error) {
	return es.svc.IdentifyPAT(ctx, paToken)
}

func (es *eventStore) AuthorizePAT(ctx context.Context, paToken string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) error {
	return es.svc.AuthorizePAT(ctx, paToken, platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
}

func (es *eventStore) CheckPAT(ctx context.Context, userID, patID string, platformEntityType auth.PlatformEntityType, optionalDomainID string, optionalDomainEntityType auth.DomainEntityType, operation auth.OperationType, entityIDs ...string) error {
	return es.svc.CheckPAT(ctx, userID, patID, platformEntityType, optionalDomainID, optionalDomainEntityType, operation, entityIDs...)
}
