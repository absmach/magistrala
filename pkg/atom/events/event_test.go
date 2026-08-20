// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawEvent is a minimal events.Event for tests: it hands back the map it was
// built with, or errData if set, mirroring what a real AMQP delivery decodes
// into before Handler ever sees it.
type rawEvent struct {
	data    map[string]any
	errData error
}

func (r rawEvent) Encode() (map[string]any, error) {
	if r.errData != nil {
		return nil, r.errData
	}
	return r.data, nil
}

func TestHandlerInvalidatesMappedFamily(t *testing.T) {
	cases := []struct {
		name   string
		event  string
		family string
	}{
		{"entity create", "entity.create", FamilyTranslation},
		{"entity update", "entity.update", FamilyTranslation},
		{"group member add", "group_member.add", FamilyAuthorizedSet},
		{"group member remove", "group_member.remove", FamilyAuthorizedSet},
		{"direct policy create", "direct_policy.create", FamilyAuthorizedSet},
		{"direct policy delete", "direct_policy.delete", FamilyAuthorizedSet},
		{"group parent set", "group.parent.set", FamilyAuthorizedSet},
		{"group parent remove", "group.parent.remove", FamilyAuthorizedSet},
		{"group delete", "group.delete", FamilyAuthorizedSet},
		{"object group add", "entity.object_group.add", FamilyAuthorizedSet},
		{"object group remove", "entity.object_group.remove", FamilyAuthorizedSet},
		{"object groups clear", "entity.object_groups.clear", FamilyAuthorizedSet},
		{"role assignment create", "role_assignment.create", FamilyAuthorizedSet},
		{"role assignment delete", "role_assignment.delete", FamilyAuthorizedSet},
		{"role permission blocks replace", "role.permission_blocks.replace", FamilyAuthorizedSet},
		{"role delete", "role.delete", FamilyAuthorizedSet},
		{"role restore", "role.restore", FamilyAuthorizedSet},
		{"role purge", "role.purge", FamilyAuthorizedSet},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := &fakeInvalidator{}
			registry := NewRegistry()
			registry.Register(tc.family, inv)

			h := NewHandler(registry, nil)
			ev := rawEvent{data: map[string]any{"event": tc.event, "tenant_id": "domain-1"}}

			require.NoError(t, h.Handle(context.Background(), ev))
			assert.Equal(t, []string{"domain-1"}, inv.keys)
		})
	}
}

// TestHandlerEntityLifecycleInvalidatesBothFamilies pins the dual-family
// mapping: entity.delete, entity.restore and entity.purge each change both
// the translation map and the authorized set (a soft delete drops the device
// from authorized_object_ids, restore brings it back, purge removes it for
// good), so a subject re-reading after any of them must recompute both halves
// of the grant.
func TestHandlerEntityLifecycleInvalidatesBothFamilies(t *testing.T) {
	for _, event := range []string{"entity.delete", "entity.restore", "entity.purge"} {
		t.Run(event, func(t *testing.T) {
			translation := &fakeInvalidator{}
			authorized := &fakeInvalidator{}
			registry := NewRegistry()
			registry.Register(FamilyTranslation, translation)
			registry.Register(FamilyAuthorizedSet, authorized)

			h := NewHandler(registry, nil)
			ev := rawEvent{data: map[string]any{"event": event, "tenant_id": "domain-1"}}

			require.NoError(t, h.Handle(context.Background(), ev))
			assert.Equal(t, []string{"domain-1"}, translation.keys)
			assert.Equal(t, []string{"domain-1"}, authorized.keys)
		})
	}
}

// atomEventVocabulary is the set of event names absmach/atom currently emits
// (src/identity/repo.rs, src/graphql/entities.rs, src/graphql/groups.rs,
// src/authz/repo.rs). It is a wire contract: this consumer must only map
// names that exist in it, so a typo or a PRD-only name -- the historical
// entity.parent_group.{set,clear} mistake -- fails the test below instead of
// silently never invalidating.
var atomEventVocabulary = map[string]struct{}{
	"entity.create": {}, "entity.update": {}, "entity.delete": {},
	"entity.restore": {}, "entity.purge": {},
	"entity.enable": {}, "entity.disable": {}, "entity.suspend": {},
	"entity.object_group.add": {}, "entity.object_group.remove": {},
	"entity.object_groups.clear": {},
	"group.create":               {}, "group.update": {}, "group.delete": {},
	"group.restore": {}, "group.purge": {},
	"group.enable": {}, "group.disable": {}, "group.suspend": {},
	"group.parent.set": {}, "group.parent.remove": {},
	"group_member.add": {}, "group_member.remove": {},
	"direct_policy.create": {}, "direct_policy.delete": {},
	"role.create":                    {},
	"role.update":                    {},
	"role.delete":                    {},
	"role.restore":                   {},
	"role.purge":                     {},
	"role.permission_blocks.replace": {},
	"role_assignment.create":         {},
	"role_assignment.delete":         {},
}

// TestEventFamiliesKeysAreRealAtomEvents guards the "dead name" failure mode:
// every event name this consumer maps must be emitted by Atom.
func TestEventFamiliesKeysAreRealAtomEvents(t *testing.T) {
	for name := range eventFamilies {
		_, ok := atomEventVocabulary[name]
		require.Truef(t, ok, "eventFamilies key %q is not an event Atom emits", name)
	}
}

// TestAuthorizationAffectingEventsAreMapped guards the "missing name" failure
// mode: every Atom event that changes a UUID's external_id or a subject's
// authorized device set must be mapped, so a gap such as group.parent.set
// being absent is caught rather than silently degrading to TTL-only
// invalidation.
func TestAuthorizationAffectingEventsAreMapped(t *testing.T) {
	needed := []string{
		"entity.create", "entity.update", "entity.delete",
		"entity.restore", "entity.purge",
		"entity.object_group.add", "entity.object_group.remove",
		"entity.object_groups.clear",
		"group.delete",
		"group.parent.set", "group.parent.remove",
		"group_member.add", "group_member.remove",
		"direct_policy.create", "direct_policy.delete",
		"role_assignment.create", "role_assignment.delete",
		"role.permission_blocks.replace",
		"role.delete", "role.restore", "role.purge",
	}

	for _, name := range needed {
		_, ok := eventFamilies[name]
		require.Truef(t, ok, "authorization-affecting Atom event %q is not mapped", name)
	}
}

func TestHandlerIgnoresUnrecognisedEvent(t *testing.T) {
	inv := &fakeInvalidator{}
	registry := NewRegistry()
	registry.Register(FamilyTranslation, inv)
	registry.Register(FamilyAuthorizedSet, inv)

	h := NewHandler(registry, nil)
	ev := rawEvent{data: map[string]any{"event": "audit.log", "tenant_id": "domain-1"}}

	require.NoError(t, h.Handle(context.Background(), ev))
	assert.Empty(t, inv.keys, "an event this consumer does not track must never invalidate anything")
}

func TestHandlerSkipsEventMissingTenantID(t *testing.T) {
	inv := &fakeInvalidator{}
	registry := NewRegistry()
	registry.Register(FamilyTranslation, inv)

	h := NewHandler(registry, nil)
	ev := rawEvent{data: map[string]any{"event": "entity.update"}}

	require.NoError(t, h.Handle(context.Background(), ev), "a missing tenant_id must not be treated as an error worth Nack-ing")
	assert.Empty(t, inv.keys)
}

func TestHandlePropagatesDecodeError(t *testing.T) {
	h := NewHandler(NewRegistry(), nil)
	wantErr := errors.New("malformed payload")

	err := h.Handle(context.Background(), rawEvent{errData: wantErr})
	assert.ErrorIs(t, err, wantErr)
}

func TestHandleDuplicateDeliveryIsHarmless(t *testing.T) {
	inv := &fakeInvalidator{}
	registry := NewRegistry()
	registry.Register(FamilyTranslation, inv)

	h := NewHandler(registry, nil)
	ev := rawEvent{data: map[string]any{"event": "entity.update", "tenant_id": "domain-1"}}

	require.NoError(t, h.Handle(context.Background(), ev))
	require.NoError(t, h.Handle(context.Background(), ev))

	assert.Equal(t, []string{"domain-1", "domain-1"}, inv.keys, "replaying the same event must not error even though it invalidates twice")
}

func TestHandleScopesInvalidationToEventsOwnTenant(t *testing.T) {
	inv := &fakeInvalidator{}
	registry := NewRegistry()
	registry.Register(FamilyTranslation, inv)

	h := NewHandler(registry, nil)

	require.NoError(t, h.Handle(context.Background(), rawEvent{data: map[string]any{"event": "entity.update", "tenant_id": "domain-1"}}))
	require.NoError(t, h.Handle(context.Background(), rawEvent{data: map[string]any{"event": "entity.update", "tenant_id": "domain-2"}}))

	assert.Equal(t, []string{"domain-1", "domain-2"}, inv.keys, "each event must only carry its own tenant into the invalidation")
}
