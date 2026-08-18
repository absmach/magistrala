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
// Names follow absmach/atom's current vocabulary (src/identity/repo.rs,
// src/graphql/entities.rs, src/graphql/groups.rs, src/authz/repo.rs). Two
// historical pitfalls to keep in mind when adding names:
//   - a group's place in the hierarchy changes via group.parent.set and
//     group.parent.remove (target_kind "group"). Atom emits no
//     "entity.parent_group.*" event -- the spelling MG-14's PRD used -- so
//     mapping that name would silently never invalidate anything.
//   - entity.object_group.{add,remove} and entity.object_groups.clear are
//     entity-to-group membership changes, a different operation from parent
//     assignment. Both change which devices a subject can read and so map to
//     FamilyAuthorizedSet.
var eventFamilies = map[string][]string{
	"entity.create": {FamilyTranslation},
	"entity.update": {FamilyTranslation},
	"entity.delete": {FamilyTranslation},
	// restore and purge bookend entity.delete but also change the authorized
	// set: restore reactivates a soft-deleted device inside its groups, and
	// purge hard-deletes it and removes its authz references outright
	// (purge_authz_references_for_ids), so both invalidate both families.
	"entity.restore": {FamilyTranslation, FamilyAuthorizedSet},
	"entity.purge":   {FamilyTranslation, FamilyAuthorizedSet},

	"group_member.add":    {FamilyAuthorizedSet},
	"group_member.remove": {FamilyAuthorizedSet},

	"direct_policy.create": {FamilyAuthorizedSet},
	"direct_policy.delete": {FamilyAuthorizedSet},

	"group.parent.set":    {FamilyAuthorizedSet},
	"group.parent.remove": {FamilyAuthorizedSet},
	// group.delete drops a whole group and every grant that flowed through it
	// (object and principal), which group_member.remove -- single-member only
	// -- does not cover.
	"group.delete": {FamilyAuthorizedSet},

	"entity.object_group.add":    {FamilyAuthorizedSet},
	"entity.object_group.remove": {FamilyAuthorizedSet},
	"entity.object_groups.clear": {FamilyAuthorizedSet},

	// Role-based grants resolve through the same subject_effective_grants
	// union as direct policies (absmach/atom migrations/001_initial.sql), so
	// granting or revoking a role -- or editing the permission blocks of one
	// that is already granted -- changes a subject's authorized device set
	// exactly like direct_policy.* does. role.create and role.update are
	// deliberately absent: a role grants nothing by itself until it has both a
	// permission block and an assignment, which the mapped events cover.
	"role_assignment.create":         {FamilyAuthorizedSet},
	"role_assignment.delete":         {FamilyAuthorizedSet},
	"role.permission_blocks.replace": {FamilyAuthorizedSet},
	"role.delete":                    {FamilyAuthorizedSet},
	"role.restore":                   {FamilyAuthorizedSet},
	"role.purge":                     {FamilyAuthorizedSet},
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
