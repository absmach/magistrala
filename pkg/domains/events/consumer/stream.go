// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/roles"
	rconsumer "github.com/absmach/supermq/pkg/roles/rolemanager/events/consumer"
)

const (
	stream = "events.supermq.domain.*"

	create              = "domain.create"
	update              = "domain.update"
	enable              = "domain.enable"
	disable             = "domain.disable"
	freeze              = "domain.freeze"
	delete              = "domain.delete"
	removeDomainMembers = "domain.members.remove"
)

var (
	errNoOperationKey           = errors.New("operation key is not found in event message")
	errCreateDomainEvent        = errors.New("failed to consume domain create event")
	errUpdateDomainEvent        = errors.New("failed to consume domain update event")
	errEnableDomainEvent        = errors.New("failed to consume domain enable event")
	errDisableDomainEvent       = errors.New("failed to consume domain disable event")
	errFreezeDomainEvent        = errors.New("failed to consume domain freeze event")
	errRemoveDomainMembersEvent = errors.New("failed to consume domain remove members event")
	errDeleteDomainEvent        = errors.New("failed to consume domain delete event")
)

type eventHandler struct {
	repo              domains.Repository
	rolesEventHandler rconsumer.EventHandler
	entityRoleManager roles.RoleManager
}

func DomainsEventsSubscribe(ctx context.Context, repo domains.Repository, roleManager roles.RoleManager, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:         stream,
		Consumer:       esConsumerName,
		Handler:        NewEventHandler(repo, roleManager),
		DeliveryPolicy: messaging.DeliverNewPolicy,
		Ordered:        true,
	}
	return subscriber.Subscribe(ctx, subConfig)
}

// NewEventHandler returns new event store handler.
func NewEventHandler(repo domains.Repository, roleManager roles.RoleManager) events.EventHandler {
	reh := rconsumer.NewEventHandler("domain", repo)
	return &eventHandler{
		repo:              repo,
		rolesEventHandler: reh,
		entityRoleManager: roleManager,
	}
}

func (es *eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	op, ok := msg["operation"]

	if !ok {
		return errNoOperationKey
	}
	switch op {
	case create:
		return es.createDomainHandler(ctx, msg)
	case update:
		return es.updateDomainHandler(ctx, msg)
	case enable:
		return es.enableDomainHandler(ctx, msg)
	case disable:
		return es.disableDomainHandler(ctx, msg)
	case freeze:
		return es.freezeDomainHandler(ctx, msg)
	case removeDomainMembers:
		return es.removeDomainMembersHandler(ctx, msg)
	case delete:
		return es.deleteDomainHandler(ctx, msg)
	}

	return es.rolesEventHandler.Handle(ctx, op, msg)
}

func (es *eventHandler) createDomainHandler(ctx context.Context, data map[string]any) error {
	d, rps, err := decodeCreateDomainEvent(data)
	if err != nil {
		return errors.Wrap(errCreateDomainEvent, err)
	}

	if _, err := es.repo.SaveDomain(ctx, d); err != nil {
		return errors.Wrap(errCreateDomainEvent, err)
	}
	if _, err := es.repo.AddRoles(ctx, rps); err != nil {
		return errors.Wrap(errCreateDomainEvent, err)
	}

	return nil
}

func (es *eventHandler) updateDomainHandler(ctx context.Context, data map[string]any) error {
	d, err := decodeUpdateDomainEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateDomainEvent, err)
	}

	if _, err := es.repo.UpdateDomain(
		ctx,
		d.ID,
		domains.DomainReq{
			Name:      &d.Name,
			Metadata:  &d.Metadata,
			Tags:      &d.Tags,
			UpdatedBy: &d.UpdatedBy,
			UpdatedAt: &d.UpdatedAt,
		},
	); err != nil {
		return errors.Wrap(errUpdateDomainEvent, err)
	}

	return nil
}

func (es *eventHandler) enableDomainHandler(ctx context.Context, data map[string]any) error {
	d, err := decodeEnableDomainEvent(data)
	if err != nil {
		return errors.Wrap(errEnableDomainEvent, err)
	}

	enabled := domains.EnabledStatus
	if _, err := es.repo.UpdateDomain(ctx, d.ID, domains.DomainReq{Status: &enabled, UpdatedBy: &d.UpdatedBy, UpdatedAt: &d.UpdatedAt}); err != nil {
		return errors.Wrap(errEnableDomainEvent, err)
	}

	return nil
}

func (es *eventHandler) disableDomainHandler(ctx context.Context, data map[string]any) error {
	d, err := decodeDisableDomainEvent(data)
	if err != nil {
		return errors.Wrap(errDisableDomainEvent, err)
	}

	disabled := domains.DisabledStatus
	if _, err := es.repo.UpdateDomain(ctx, d.ID, domains.DomainReq{Status: &disabled, UpdatedBy: &d.UpdatedBy, UpdatedAt: &d.UpdatedAt}); err != nil {
		return errors.Wrap(errDisableDomainEvent, err)
	}

	return nil
}

func (es *eventHandler) freezeDomainHandler(ctx context.Context, data map[string]any) error {
	d, err := decodeFreezeDomainEvent(data)
	if err != nil {
		return errors.Wrap(errFreezeDomainEvent, err)
	}

	freeze := domains.FreezeStatus
	if _, err := es.repo.UpdateDomain(ctx, d.ID, domains.DomainReq{Status: &freeze, UpdatedBy: &d.UpdatedBy, UpdatedAt: &d.UpdatedAt}); err != nil {
		return errors.Wrap(errFreezeDomainEvent, err)
	}

	return nil
}

func (es *eventHandler) removeDomainMembersHandler(ctx context.Context, data map[string]any) error {
	domainID, memberIDs, err := decodeRemoveDomainMembersEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveDomainMembersEvent, err)
	}

	if err := es.repo.RemoveEntityMembers(ctx, domainID, memberIDs); err != nil {
		return errors.Wrap(errRemoveDomainMembersEvent, err)
	}

	for _, memberID := range memberIDs {
		if err := es.entityRoleManager.RemoveMemberFromDomain(ctx, domainID, memberID); err != nil {
			return errors.Wrap(errRemoveDomainMembersEvent, err)
		}
	}

	return nil
}

func (es *eventHandler) deleteDomainHandler(ctx context.Context, data map[string]any) error {
	d, err := decodeDeleteDomainEvent(data)
	if err != nil {
		return errors.Wrap(errDeleteDomainEvent, err)
	}

	if err := es.repo.DeleteDomain(ctx, d.ID); err != nil {
		return errors.Wrap(errDeleteDomainEvent, err)
	}

	return nil
}
