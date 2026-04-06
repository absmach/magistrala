// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package consumer

import (
	"context"
	"log/slog"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/events"
	"github.com/absmach/magistrala/pkg/events/store"
	rconsumer "github.com/absmach/magistrala/pkg/roles/rolemanager/events/consumer"
	"github.com/absmach/magistrala/re"
)

const (
	stream = "events.magistrala.rule.*"

	create         = "rule.create"
	update         = "rule.update"
	updateTags     = "rule.update_tags"
	updateSchedule = "rule.update_schedule"
	enable         = "rule.enable"
	disable        = "rule.disable"
	remove         = "rule.remove"
)

var (
	errNoOperationKey          = errors.New("operation key is not found in event message")
	errAddRuleEvent            = errors.New("failed to consume rule create event")
	errUpdateRuleEvent         = errors.New("failed to consume rule update event")
	errUpdateRuleTagsEvent     = errors.New("failed to consume rule update tags event")
	errUpdateRuleScheduleEvent = errors.New("failed to consume rule update schedule event")
	errEnableRuleEvent         = errors.New("failed to consume rule enable event")
	errDisableRuleEvent        = errors.New("failed to consume rule disable event")
	errRemoveRuleEvent         = errors.New("failed to consume rule remove event")
)

type eventHandler struct {
	repo              re.Repository
	rolesEventHandler rconsumer.EventHandler
}

func RulesEventsSubscribe(ctx context.Context, repo re.Repository, esURL, esConsumerName string, logger *slog.Logger) error {
	subscriber, err := store.NewSubscriber(ctx, esURL, "re-es-sub", logger)
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
func NewEventHandler(repo re.Repository) events.EventHandler {
	reh := rconsumer.NewEventHandler("rule", repo)
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
		return es.addRuleHandler(ctx, msg)
	case update:
		return es.updateRuleHandler(ctx, msg)
	case updateTags:
		return es.updateRuleTagsHandler(ctx, msg)
	case updateSchedule:
		return es.updateRuleScheduleHandler(ctx, msg)
	case enable:
		return es.enableRuleHandler(ctx, msg)
	case disable:
		return es.disableRuleHandler(ctx, msg)
	case remove:
		return es.removeRuleHandler(ctx, msg)
	}

	return es.rolesEventHandler.Handle(ctx, op, msg)
}

func (es *eventHandler) addRuleHandler(ctx context.Context, data map[string]any) error {
	r, rps, err := decodeAddRuleEvent(data)
	if err != nil {
		return errors.Wrap(errAddRuleEvent, err)
	}

	if _, err := es.repo.AddRule(ctx, r); err != nil {
		return errors.Wrap(errAddRuleEvent, err)
	}

	if _, err := es.repo.AddRoles(ctx, rps); err != nil {
		return errors.Wrap(errAddRuleEvent, err)
	}

	return nil
}

func (es *eventHandler) updateRuleHandler(ctx context.Context, data map[string]any) error {
	r, err := decodeUpdateRuleEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateRuleEvent, err)
	}

	if _, err := es.repo.UpdateRule(ctx, r); err != nil {
		return errors.Wrap(errUpdateRuleEvent, err)
	}

	return nil
}

func (es *eventHandler) updateRuleTagsHandler(ctx context.Context, data map[string]any) error {
	r, err := decodeUpdateRuleTagsEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateRuleTagsEvent, err)
	}

	if _, err := es.repo.UpdateRuleTags(ctx, r); err != nil {
		return errors.Wrap(errUpdateRuleTagsEvent, err)
	}

	return nil
}

func (es *eventHandler) updateRuleScheduleHandler(ctx context.Context, data map[string]any) error {
	r, err := decodeUpdateRuleScheduleEvent(data)
	if err != nil {
		return errors.Wrap(errUpdateRuleScheduleEvent, err)
	}

	if _, err := es.repo.UpdateRuleSchedule(ctx, r); err != nil {
		return errors.Wrap(errUpdateRuleScheduleEvent, err)
	}

	return nil
}

func (es *eventHandler) enableRuleHandler(ctx context.Context, data map[string]any) error {
	r, err := decodeEnableRuleEvent(data)
	if err != nil {
		return errors.Wrap(errEnableRuleEvent, err)
	}

	if _, err := es.repo.UpdateRuleStatus(ctx, r); err != nil {
		return errors.Wrap(errEnableRuleEvent, err)
	}

	return nil
}

func (es *eventHandler) disableRuleHandler(ctx context.Context, data map[string]any) error {
	r, err := decodeDisableRuleEvent(data)
	if err != nil {
		return errors.Wrap(errDisableRuleEvent, err)
	}

	if _, err := es.repo.UpdateRuleStatus(ctx, r); err != nil {
		return errors.Wrap(errDisableRuleEvent, err)
	}

	return nil
}

func (es *eventHandler) removeRuleHandler(ctx context.Context, data map[string]any) error {
	id, err := decodeRemoveRuleEvent(data)
	if err != nil {
		return errors.Wrap(errRemoveRuleEvent, err)
	}

	if err := es.repo.RemoveRule(ctx, id); err != nil {
		return errors.Wrap(errRemoveRuleEvent, err)
	}

	return nil
}
