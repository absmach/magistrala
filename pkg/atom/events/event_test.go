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
		{"entity delete", "entity.delete", FamilyTranslation},
		{"group member add", "group_member.add", FamilyAuthorizedSet},
		{"group member remove", "group_member.remove", FamilyAuthorizedSet},
		{"direct policy create", "direct_policy.create", FamilyAuthorizedSet},
		{"direct policy delete", "direct_policy.delete", FamilyAuthorizedSet},
		{"parent group set (PRD spelling)", "entity.parent_group.set", FamilyAuthorizedSet},
		{"parent group clear (PRD spelling)", "entity.parent_group.clear", FamilyAuthorizedSet},
		{"object group add (Atom's actual spelling)", "entity.object_group.add", FamilyAuthorizedSet},
		{"object group remove (Atom's actual spelling)", "entity.object_group.remove", FamilyAuthorizedSet},
		{"object groups clear (Atom's actual spelling)", "entity.object_groups.clear", FamilyAuthorizedSet},
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
