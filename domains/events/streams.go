// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/roles"
	rmEvents "github.com/absmach/supermq/pkg/roles/rolemanager/events"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	supermqPrefix               = "supermq."
	createStream                = supermqPrefix + domainCreate
	retrieveStream              = supermqPrefix + domainRetrieve
	updateStream                = supermqPrefix + domainUpdate
	enableStream                = supermqPrefix + domainEnable
	disableStream               = supermqPrefix + domainDisable
	freezeStream                = supermqPrefix + domainFreeze
	listStream                  = supermqPrefix + domainList
	deleteStream                = supermqPrefix + domainDelete
	sendInvitationStream        = supermqPrefix + invitationSend
	acceptInvitationStream      = supermqPrefix + invitationAccept
	rejectInvitationStream      = supermqPrefix + invitationReject
	listInvitationsStream       = supermqPrefix + invitationList
	listDomainInvitationsStream = supermqPrefix + invitationListDomain
	deleteInvitationStream      = supermqPrefix + invitationDelete
)

var _ domains.Service = (*eventStore)(nil)

type eventStore struct {
	events.Publisher
	svc domains.Service
	rmEvents.RoleManagerEventStore
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(ctx context.Context, svc domains.Service, url string) (domains.Service, error) {
	publisher, err := store.NewPublisher(ctx, url)
	if err != nil {
		return nil, err
	}

	res := rmEvents.NewRoleManagerEventStore("domains", domainPrefix, svc, publisher)

	return &eventStore{
		svc:                   svc,
		Publisher:             publisher,
		RoleManagerEventStore: res,
	}, nil
}

func (es *eventStore) CreateDomain(ctx context.Context, session authn.Session, domain domains.Domain) (domains.Domain, []roles.RoleProvision, error) {
	domain, rps, err := es.svc.CreateDomain(ctx, session, domain)
	if err != nil {
		return domain, rps, err
	}

	event := createDomainEvent{
		Domain:           domain,
		rolesProvisioned: rps,
		Session:          session,
		requestID:        middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, createStream, event); err != nil {
		return domain, rps, err
	}

	return domain, rps, nil
}

func (es *eventStore) RetrieveDomain(ctx context.Context, session authn.Session, id string, withRoles bool) (domains.Domain, error) {
	domain, err := es.svc.RetrieveDomain(ctx, session, id, withRoles)
	if err != nil {
		return domain, err
	}

	event := retrieveDomainEvent{
		domain,
		session,
		middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, retrieveStream, event); err != nil {
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
		domain:    domain,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, updateStream, event); err != nil {
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
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, enableStream, event); err != nil {
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
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, disableStream, event); err != nil {
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
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, freezeStream, event); err != nil {
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
		Page:       p,
		total:      dp.Total,
		userID:     session.UserID,
		tokenType:  session.Type.String(),
		superAdmin: session.SuperAdmin,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listStream, event); err != nil {
		return dp, err
	}

	return dp, nil
}

func (es *eventStore) DeleteDomain(ctx context.Context, session authn.Session, id string) error {
	if err := es.svc.DeleteDomain(ctx, session, id); err != nil {
		return err
	}

	event := deleteDomainEvent{
		domainID:  id,
		Session:   session,
		requestID: middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, deleteStream, event)
}

func (es *eventStore) SendInvitation(ctx context.Context, session authn.Session, invitation domains.Invitation) (domains.Invitation, error) {
	inv, err := es.svc.SendInvitation(ctx, session, invitation)
	if err != nil {
		return domains.Invitation{}, err
	}

	event := sendInvitationEvent{
		invitation: inv,
		session:    session,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, sendInvitationStream, event); err != nil {
		return inv, err
	}

	return inv, nil
}

func (es *eventStore) ListInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (domains.InvitationPage, error) {
	ip, err := es.svc.ListInvitations(ctx, session, pm)
	if err != nil {
		return ip, err
	}

	event := listInvitationsEvent{
		InvitationPageMeta: pm,
		session:            session,
		requestID:          middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listInvitationsStream, event); err != nil {
		return ip, err
	}

	return ip, nil
}

func (es *eventStore) ListDomainInvitations(ctx context.Context, session authn.Session, pm domains.InvitationPageMeta) (domains.InvitationPage, error) {
	ip, err := es.svc.ListDomainInvitations(ctx, session, pm)
	if err != nil {
		return ip, err
	}

	event := listDomainInvitationsEvent{
		InvitationPageMeta: pm,
		session:            session,
		requestID:          middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, listDomainInvitationsStream, event); err != nil {
		return ip, err
	}

	return ip, nil
}

func (es *eventStore) AcceptInvitation(ctx context.Context, session authn.Session, domainID string) (domains.Invitation, error) {
	inv, err := es.svc.AcceptInvitation(ctx, session, domainID)
	if err != nil {
		return inv, err
	}

	if err := es.RoleManagerEventStore.RoleAddMembersEventPublisher(ctx, inv.DomainID, inv.RoleID, []string{inv.InviteeUserID}); err != nil {
		return inv, err
	}

	event := acceptInvitationEvent{
		invitation: inv,
		session:    session,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, acceptInvitationStream, event); err != nil {
		return inv, err
	}
	return inv, nil
}

func (es *eventStore) RejectInvitation(ctx context.Context, session authn.Session, domainID string) (domains.Invitation, error) {
	inv, err := es.svc.RejectInvitation(ctx, session, domainID)
	if err != nil {
		return domains.Invitation{}, err
	}

	event := rejectInvitationEvent{
		invitation: inv,
		session:    session,
		requestID:  middleware.GetReqID(ctx),
	}

	if err := es.Publish(ctx, rejectInvitationStream, event); err != nil {
		return inv, err
	}

	return inv, nil
}

func (es *eventStore) DeleteInvitation(ctx context.Context, session authn.Session, inviteeUserID, domainID string) error {
	if err := es.svc.DeleteInvitation(ctx, session, inviteeUserID, domainID); err != nil {
		return err
	}

	event := deleteInvitationEvent{
		inviteeUserID: inviteeUserID,
		domainID:      domainID,
		session:       session,
		requestID:     middleware.GetReqID(ctx),
	}

	return es.Publish(ctx, deleteInvitationStream, event)
}
