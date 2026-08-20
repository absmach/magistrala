// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"testing"
	"time"
)

func TestTenantFromFields(t *testing.T) {
	created := time.Date(2026, 4, 30, 10, 11, 12, 0, time.UTC)
	got := TenantFromFields(ObjectFields{
		ID:        "domain-1",
		Name:      "Acme",
		Route:     "acme",
		Status:    "enabled",
		Tags:      []string{"prod"},
		Metadata:  map[string]any{"tier": "gold"},
		CreatedBy: "user-1",
		CreatedAt: created,
	})

	if got.ID != "domain-1" || got.Name != "Acme" || got.Route != "acme" {
		t.Fatalf("unexpected tenant: %+v", got)
	}
	if got.Attributes["source"] != "magistrala" {
		t.Fatalf("missing source attribute: %+v", got.Attributes)
	}
	if got.Attributes["created_at"] != "2026-04-30T10:11:12Z" {
		t.Fatalf("unexpected created_at: %v", got.Attributes["created_at"])
	}
}

func TestEntityFromFields(t *testing.T) {
	got := EntityFromFields(ObjectFields{
		ID:       "device-1",
		Kind:     KindDevice,
		Name:     "pump",
		TenantID: "domain-1",
		Status:   "enabled",
		ParentID: "group-1",
		Tags:     []string{"field"},
	})

	if got.Kind != "device" || got.TenantID != "domain-1" {
		t.Fatalf("unexpected entity: %+v", got)
	}
	if got.Attributes["magistrala_kind"] != KindDevice {
		t.Fatalf("missing magistrala kind: %+v", got.Attributes)
	}
	if _, ok := got.Attributes["parent_group_id"]; ok {
		t.Fatalf("parent_group_id is a rejected Atom attribute and must not be set: %+v", got.Attributes)
	}
}

func TestDeviceKindNamesMapToAtomDeviceKind(t *testing.T) {
	for _, kind := range []string{KindDevice} {
		if got := entityKind(kind); got != atomKindDevice {
			t.Fatalf("%q must map to the atom kind %q, got %q", kind, atomKindDevice, got)
		}
	}
}

func TestResourceFromFieldsOmitsEmptyValues(t *testing.T) {
	got := ResourceFromFields(ObjectFields{
		ID:       "channel-1",
		Kind:     KindChannel,
		Name:     "telemetry",
		TenantID: "domain-1",
	})

	if _, ok := got.Attributes["tags"]; ok {
		t.Fatalf("empty tags should be omitted: %+v", got.Attributes)
	}
	if got.Attributes["source"] != "magistrala" {
		t.Fatalf("missing source attribute: %+v", got.Attributes)
	}
}

func TestResourceFromFieldsExcludesParentGroupID(t *testing.T) {
	got := ResourceFromFields(ObjectFields{
		ID:       "channel-1",
		Kind:     KindChannel,
		Name:     "telemetry",
		TenantID: "domain-1",
		ParentID: "group-1",
	})

	if _, ok := got.Attributes["parent_group_id"]; ok {
		t.Fatalf("parent_group_id is a rejected Atom attribute and must not be set: %+v", got.Attributes)
	}
}
