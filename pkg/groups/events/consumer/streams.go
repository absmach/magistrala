// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/supermq/groups"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/absmach/supermq/pkg/events"
	"github.com/absmach/supermq/pkg/events/store"
	rconsumer "github.com/absmach/supermq/pkg/roles/rolemanager/events/consumer"
)

const (
	stream = "events.supermq.groups"

	create                  = "group.create"
	update                  = "group.update"
	changeStatus            = "group.change_status"
	remove                  = "group.remove"
	addParentGroup          = "group.add_parent_group"
	removeParentGroup       = "group.remove_parent_group"
	addChildrenGroups       = "group.add_children_groups"
	removeChildrenGroups    = "group.remove_children_groups"
	removeAllChildrenGroups = "group.remove_all_children_groups"
)

var (
	errNoOperationKey              = errors.New("operation key is not found in event message")
	errCreateGroupEvent            = errors.New("failed to consume group create event")
	errUpdateGroupEvent            = errors.New("failed to consume group update event")
	errChangeStatusGroupEvent      = errors.New("failed to consume group change status event")
	errRemoveGroupEvent            = errors.New("failed to consume group remove event")
	errAddParentGroupEvent         = errors.New("failed to consume group add parent group event")
	errRemoveParentGroupEvent      = errors.New("failed to consume group remove parent group event")
	errAddChildrenGroupEvent       = errors.New("failed to consume group add children groups event")
	errRemoveChildrenGroupEvent    = errors.New("failed to consume group remove children groups event")
	errRemoveAllChildrenGroupEvent = errors.New("failed to consume group remove all children groups event")
)

type eventHandler struct {
	repo              groups.Repository
	rolesEventHandler rconsumer.EventHandler
}

func GroupsEventsSubscribe(ctx context.Context, repo groups.Repository, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, logger)
	if err != nil {
		return err
	}

	subConfig := events.SubscriberConfig{
		Stream:   stream,
		Consumer: esConsumerName,
		Handler:  NewEventHandler(repo),
		Ordered:  true,
	}
	return subscriber.Subscribe(ctx, subConfig)
}

// NewEventHandler returns new event store handler.
func NewEventHandler(repo groups.Repository) events.EventHandler {
	reh := rconsumer.NewEventHandler("group", repo)
	return &eventHandler{
		repo:              repo,
		rolesEventHandler: reh,
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
		return es.createGroupHandler(ctx, msg)
	case update:
		return es.updateGroupHandler(ctx, msg)
	case changeStatus:
		return es.changeStatusGroupHandler(ctx, msg)
	case remove:
		return es.removeGroupHandler(ctx, msg)
	case addParentGroup:
		return es.addParentGroupHandler(ctx, msg)
	case removeParentGroup:
		return es.removeParentGroupHandler(ctx, msg)
	case addChildrenGroups:
		return es.addChildrenGroupsHandler(ctx, msg)
	case removeChildrenGroups:
		return es.removeChildrenGroupsHandler(ctx, msg)
	case removeAllChildrenGroups:
		return es.removeAllChildrenGroupsHandler(ctx, msg)
	}

	return es.rolesEventHandler.Handle(ctx, op, msg)
}

func (es *eventHandler) createGroupHandler(ctx context.Context, data map[string]interface{}) error {
	g, rps, err := decodeCreateGroupEvent(data)
	if err != nil {
		return errors.Wrap(errCreateGroupEvent, err)
	}

	if _, err := es.repo.Save(ctx, g); err != nil {
		return errors.Wrap(errCreateGroupEvent, err)
	}
	if _, err := es.repo.AddRoles(ctx, rps); err != nil {
		return errors.Wrap(errCreateGroupEvent, err)
	}

	return nil
}

func (es *eventHandler) updateGroupHandler(ctx context.Context, data map[string]interface{}) error {
	g, err := decodeUpdateGroupEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateGroupEvent, err)
	}

	if _, err := es.repo.Update(ctx, g); err != nil {
		return errors.Wrap(errUpdateGroupEvent, err)
	}

	return nil
}

func (es *eventHandler) changeStatusGroupHandler(ctx context.Context, data map[string]interface{}) error {
	g, err := decodeChangeStatusGroupEvent(data)
	if err != nil {
		return errors.Wrap(errChangeStatusGroupEvent, err)
	}

	if _, err := es.repo.ChangeStatus(ctx, g); err != nil {
		return errors.Wrap(errChangeStatusGroupEvent, err)
	}

	return nil
}

func (es *eventHandler) removeGroupHandler(ctx context.Context, data map[string]interface{}) error {
	g, err := decodeRemoveGroupEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveGroupEvent, err)
	}

	if err := es.repo.Delete(ctx, g.ID); err != nil {
		return errors.Wrap(errRemoveGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) addParentGroupHandler(ctx context.Context, data map[string]interface{}) error {
	id, parent, err := decodeAddParentGroupEvent(data)
	if err != nil {
		return errors.Wrap(errAddParentGroupEvent, err)
	}
	if err := es.repo.AssignParentGroup(ctx, parent, id); err != nil {
		return errors.Wrap(errAddParentGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) removeParentGroupHandler(ctx context.Context, data map[string]interface{}) error {
	id, err := decodeRemoveParentGroupEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveParentGroupEvent, err)
	}
	g, err := es.repo.RetrieveByID(ctx, id)
	if err != nil {
		return errors.Wrap(errRemoveParentGroupEvent, err)
	}
	if err := es.repo.UnassignParentGroup(ctx, g.Parent, id); err != nil {
		return errors.Wrap(errRemoveParentGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) addChildrenGroupsHandler(ctx context.Context, data map[string]interface{}) error {
	id, cids, err := decodeAddChildrenGroupEvent(data)
	if err != nil {
		return errors.Wrap(errAddChildrenGroupEvent, err)
	}

	if err := es.repo.AssignParentGroup(ctx, id, cids...); err != nil {
		return errors.Wrap(errAddChildrenGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) removeChildrenGroupsHandler(ctx context.Context, data map[string]interface{}) error {
	id, cids, err := decodeRemoveChildrenGroupEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveChildrenGroupEvent, err)
	}

	if err := es.repo.UnassignParentGroup(ctx, id, cids...); err != nil {
		return errors.Wrap(errRemoveChildrenGroupEvent, err)
	}
	return nil
}

func (es *eventHandler) removeAllChildrenGroupsHandler(ctx context.Context, data map[string]interface{}) error {
	id, err := decodeRemoveAllChildrenGroupEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveAllChildrenGroupEvent, err)
	}
	if err := es.repo.UnassignAllChildrenGroups(ctx, id); err != nil && err != repoerr.ErrNotFound {
		return errors.Wrap(errRemoveAllChildrenGroupEvent, err)
	}
	return nil
}
