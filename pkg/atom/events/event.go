// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"log/slog"

	"github.com/absmach/magistrala/pkg/events"
)

// Cache families an Atom event can invalidate. MG-08's caches
// (readers/api/http/readauthz.go) register against both today, since they
// currently share one combined cache entry per (domain, subject); a future
// consumer with genuinely separate caches can register against just one.
const (
	// FamilyTranslation covers Atom's UUID -> external_id (device serial)
	// resolution -- ATOM-06, pkg/atom.Client.EntityExternalIDs.
	FamilyTranslation = "translation"
	// FamilyAuthorizedSet covers which devices a subject is authorized to
	// read -- MG-08's read-authorization grant cache.
	FamilyAuthorizedSet = "authorized_set"
)

// eventFamilies maps the Atom domain-event names that matter for cache
// invalidation (MG-14's PRD) to the families they invalidate. An event name
// with no entry here is ignored: Atom emits roughly 40 domain events in
// total (audit, purge, profile/version management, ...) and this consumer
// only cares about the handful that change a UUID's external_id or a
// subject's authorized device set.
//
// entity.object_group.{add,remove} and entity.object_groups.clear are
// Atom's current event names (absmach/atom src/identity/repo.rs) for what
// MG-14's PRD calls entity.parent_group.{set,clear}; both spellings are
// mapped so this keeps working across that naming.
var eventFamilies = map[string][]string{
	"entity.create": {FamilyTranslation},
	"entity.update": {FamilyTranslation},
	"entity.delete": {FamilyTranslation},

	"group_member.add":    {FamilyAuthorizedSet},
	"group_member.remove": {FamilyAuthorizedSet},

	"direct_policy.create": {FamilyAuthorizedSet},
	"direct_policy.delete": {FamilyAuthorizedSet},

	"entity.parent_group.set":   {FamilyAuthorizedSet},
	"entity.parent_group.clear": {FamilyAuthorizedSet},

	"entity.object_group.add":    {FamilyAuthorizedSet},
	"entity.object_group.remove": {FamilyAuthorizedSet},
	"entity.object_groups.clear": {FamilyAuthorizedSet},
}

// Handler adapts Atom's domain-event wire format (DomainEventPayload in
// absmach/atom src/events/mod.rs, delivered here as JSON) into Registry
// invalidations. It is the events.EventHandler that
// pkg/events/fluxmq.NewQueueSubscriber calls once per message.
type Handler struct {
	registry *Registry
	logger   *slog.Logger
}

var _ events.EventHandler = (*Handler)(nil)

// NewHandler returns a Handler dispatching through registry. A nil logger
// falls back to slog's default logger, matching pkg/events/fluxmq.
func NewHandler(registry *Registry, logger *slog.Logger) *Handler {
	return &Handler{registry: registry, logger: logger}
}

// Handle decodes one Atom domain event and invalidates every family it maps
// to, scoped to the event's own tenant_id. Atom's tenant is Magistrala's
// domain -- pkg/atom/policy_service.go binds TenantID: pr.Domain on every
// call it makes -- so tenant_id is exactly the scope a cache keyed by
// domain needs to clear, and clearing by domain rather than by a narrower
// key (e.g. one specific subject) is deliberate: several of the events this
// package tracks (direct_policy.create/delete in particular) carry no
// subject at all in their payload, only the policy row's own id, so a
// domain-wide invalidation is the coarsest -- and only reliably correct --
// granularity available without inspecting payload content the invalidate-
// only design already refuses to trust.
//
// Handle never inspects any other field: not the actor, not target_id, not
// details. Only "something in this family changed for this tenant"
// survives the decode.
//
// An event this consumer does not recognise, or one missing tenant_id, is
// not an error: Atom emits far more event types than the handful this
// package tracks, and failing on one we deliberately do not handle would
// Nack it under at-least-once delivery and wedge the queue retrying
// something that will never succeed differently.
func (h *Handler) Handle(ctx context.Context, ev events.Event) error {
	data, err := ev.Encode()
	if err != nil {
		return err
	}

	name := events.Read(data, "event", "")
	families, ok := eventFamilies[name]
	if !ok {
		return nil
	}

	tenantID := events.Read(data, "tenant_id", "")
	if tenantID == "" {
		h.logWarn("Atom event missing tenant_id, skipping invalidation", "event", name)
		return nil
	}

	var errs []error
	for _, family := range families {
		if err := h.registry.Invalidate(ctx, family, tenantID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (h *Handler) logWarn(msg string, args ...any) {
	if h.logger != nil {
		h.logger.Warn(msg, args...)
		return
	}

	slog.Warn(msg, args...)
}
