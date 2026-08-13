// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

// fakeAtomDeviceTypes stands in for the slice of Atom device types depend on:
// profile and profile-version storage, and the entity write path that
// validates attributes against a bound version.
//
// It reproduces Atom's resolution rules rather than a convenient
// approximation, because the rules are what these tests are about:
//
//   - a device type must be active to accept a new binding
//     (src/identity/repo.rs, resolve_entity_profile)
//   - with no version named, the highest-numbered *active* version is chosen
//   - with a version named, Atom binds it without checking its status — the
//     gap BindDeviceType closes on the client side
//   - an update validates against the entity's already-bound version, not
//     against a version supplied in the same call
//   - a missing binding on update coalesces to the stored one
type fakeAtomDeviceTypes struct {
	t           *testing.T
	deviceTypes map[string]*fakeDeviceType
	versions    map[string]*fakeDeviceTypeVersion
	entities    map[string]*fakeEntity
	listQueries []map[string]any
	sequence    int
}

type fakeDeviceType struct {
	ID          string
	TenantID    string
	ObjectKind  string
	Kind        string
	Key         string
	Name        string
	Description string
	Status      string
}

type fakeDeviceTypeVersion struct {
	ID           string
	DeviceTypeID string
	Version      int
	JSONSchema   map[string]any
	UISchema     map[string]any
	Status       string
}

type fakeEntity struct {
	ID                  string
	Kind                string
	Name                string
	TenantID            string
	DeviceTypeID        string
	DeviceTypeVersionID string
	Status              string
	Attributes          map[string]any
}

func newFakeAtomDeviceTypes(t *testing.T) (*fakeAtomDeviceTypes, *Client) {
	t.Helper()
	fa := &fakeAtomDeviceTypes{
		t:           t,
		deviceTypes: map[string]*fakeDeviceType{},
		versions:    map[string]*fakeDeviceTypeVersion{},
		entities:    map[string]*fakeEntity{},
	}
	srv := httptest.NewServer(http.HandlerFunc(fa.handle))
	t.Cleanup(srv.Close)
	return fa, NewClient(Config{URL: srv.URL, Timeout: time.Second})
}

func (fa *fakeAtomDeviceTypes) id(prefix string) string {
	fa.sequence++
	return fmt.Sprintf("%s-%d", prefix, fa.sequence)
}

func (fa *fakeAtomDeviceTypes) handle(w http.ResponseWriter, r *http.Request) {
	payload := decodePayload(fa.t, r)
	switch {
	case strings.Contains(payload.Query, "mutation CreateDeviceTypeVersion"):
		fa.createVersion(w, payload)
	case strings.Contains(payload.Query, "query DeviceTypeVersions("):
		fa.listVersions(w, payload)
	case strings.Contains(payload.Query, "mutation CreateDeviceType"):
		fa.createDeviceType(w, payload)
	case strings.Contains(payload.Query, "mutation UpdateDeviceType"):
		fa.updateDeviceType(w, payload)
	case strings.Contains(payload.Query, "query DeviceTypes("):
		fa.listDeviceTypes(w, payload)
	case strings.Contains(payload.Query, "query DeviceType("):
		fa.getDeviceType(w, payload)
	case strings.Contains(payload.Query, "mutation CreateEntity"):
		fa.createEntity(w, payload)
	case strings.Contains(payload.Query, "mutation UpdateEntity"):
		fa.updateEntity(w, payload)
	case strings.Contains(payload.Query, "query Entities("):
		fa.listEntities(w, payload)
	case strings.Contains(payload.Query, "query Entity("):
		fa.getEntity(w, payload)
	default:
		fa.t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
	}
}

func (fa *fakeAtomDeviceTypes) fail(w http.ResponseWriter, format string, args ...any) {
	_ = writeJSON(w, map[string]any{
		"errors": []map[string]string{{"message": fmt.Sprintf(format, args...)}},
	})
}

func (fa *fakeAtomDeviceTypes) createDeviceType(w http.ResponseWriter, payload gqlPayload) {
	input, _ := payload.Variables["input"].(map[string]any)
	deviceType := &fakeDeviceType{
		ID:          fa.id("dt"),
		TenantID:    stringField(input, "tenantId"),
		ObjectKind:  stringField(input, "objectKind"),
		Kind:        stringField(input, "kind"),
		Key:         stringField(input, "key"),
		Name:        stringField(input, "displayName"),
		Description: stringField(input, "description"),
		Status:      stringField(input, "status"),
	}
	if deviceType.Status == "" {
		deviceType.Status = DeviceTypeStatusActive
	}
	for _, existing := range fa.deviceTypes {
		if existing.TenantID == deviceType.TenantID && existing.Key == deviceType.Key {
			fa.fail(w, "already exists")
			return
		}
	}
	fa.deviceTypes[deviceType.ID] = deviceType
	_ = writeJSON(w, map[string]any{"data": map[string]any{"createProfile": deviceTypeJSON(deviceType)}})
}

func (fa *fakeAtomDeviceTypes) getDeviceType(w http.ResponseWriter, payload gqlPayload) {
	id, _ := payload.Variables["id"].(string)
	deviceType, ok := fa.deviceTypes[id]
	if !ok {
		fa.fail(w, "profile %s not found", id)
		return
	}
	_ = writeJSON(w, map[string]any{"data": map[string]any{"profile": deviceTypeJSON(deviceType)}})
}

func (fa *fakeAtomDeviceTypes) listDeviceTypes(w http.ResponseWriter, payload gqlPayload) {
	fa.listQueries = append(fa.listQueries, payload.Variables)

	tenantID, hasTenant := payload.Variables["tenantId"].(string)
	objectKind, _ := payload.Variables["objectKind"].(string)
	kind, _ := payload.Variables["kind"].(string)
	status, _ := payload.Variables["status"].(string)

	var items []map[string]any
	for _, deviceType := range fa.deviceTypes {
		// Atom filters on tenant only when one is given; no tenant means every
		// tenant's types, which is why this client always sends one.
		if hasTenant && deviceType.TenantID != tenantID {
			continue
		}
		if objectKind != "" && deviceType.ObjectKind != objectKind {
			continue
		}
		if kind != "" && deviceType.Kind != kind {
			continue
		}
		if status != "" && deviceType.Status != status {
			continue
		}
		items = append(items, deviceTypeJSON(deviceType))
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["key"].(string) < items[j]["key"].(string) })

	_ = writeJSON(w, map[string]any{
		"data": map[string]any{"profiles": map[string]any{"items": items, "total": len(items)}},
	})
}

func (fa *fakeAtomDeviceTypes) updateDeviceType(w http.ResponseWriter, payload gqlPayload) {
	id, _ := payload.Variables["id"].(string)
	deviceType, ok := fa.deviceTypes[id]
	if !ok {
		fa.fail(w, "profile %s not found", id)
		return
	}
	input, _ := payload.Variables["input"].(map[string]any)
	if name := stringField(input, "displayName"); name != "" {
		deviceType.Name = name
	}
	if description := stringField(input, "description"); description != "" {
		deviceType.Description = description
	}
	if status := stringField(input, "status"); status != "" {
		deviceType.Status = status
	}
	_ = writeJSON(w, map[string]any{"data": map[string]any{"updateProfile": deviceTypeJSON(deviceType)}})
}

func (fa *fakeAtomDeviceTypes) createVersion(w http.ResponseWriter, payload gqlPayload) {
	deviceTypeID, _ := payload.Variables["profileId"].(string)
	if _, ok := fa.deviceTypes[deviceTypeID]; !ok {
		fa.fail(w, "profile %s not found", deviceTypeID)
		return
	}
	input, _ := payload.Variables["input"].(map[string]any)
	number := 0
	if raw, ok := input["version"].(float64); ok {
		number = int(raw)
	}
	for _, existing := range fa.versions {
		if existing.DeviceTypeID == deviceTypeID && existing.Version == number {
			fa.fail(w, "already exists")
			return
		}
	}

	version := &fakeDeviceTypeVersion{
		ID:           fa.id("dtv"),
		DeviceTypeID: deviceTypeID,
		Version:      number,
		Status:       stringField(input, "status"),
	}
	version.JSONSchema, _ = input["jsonSchema"].(map[string]any)
	version.UISchema, _ = input["uiSchema"].(map[string]any)
	if version.Status == "" {
		version.Status = DeviceTypeVersionStatusActive
	}
	fa.versions[version.ID] = version
	_ = writeJSON(w, map[string]any{"data": map[string]any{"createProfileVersion": versionJSON(version)}})
}

func (fa *fakeAtomDeviceTypes) listVersions(w http.ResponseWriter, payload gqlPayload) {
	deviceTypeID, _ := payload.Variables["profileId"].(string)
	var items []map[string]any
	for _, version := range fa.versionsOf(deviceTypeID) {
		items = append(items, versionJSON(version))
	}
	_ = writeJSON(w, map[string]any{"data": map[string]any{"profileVersions": items}})
}

// versionsOf returns a device type's versions, newest last.
func (fa *fakeAtomDeviceTypes) versionsOf(deviceTypeID string) []*fakeDeviceTypeVersion {
	var versions []*fakeDeviceTypeVersion
	for _, version := range fa.versions {
		if version.DeviceTypeID == deviceTypeID {
			versions = append(versions, version)
		}
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i].Version < versions[j].Version })
	return versions
}

// resolveBinding mirrors Atom's resolve_entity_profile.
func (fa *fakeAtomDeviceTypes) resolveBinding(deviceTypeID, versionID string) (*fakeDeviceTypeVersion, error) {
	deviceType, ok := fa.deviceTypes[deviceTypeID]
	if !ok {
		return nil, fmt.Errorf("profile %s not found", deviceTypeID)
	}
	if deviceType.Status != DeviceTypeStatusActive {
		return nil, fmt.Errorf("profile %s is not active", deviceTypeID)
	}

	if versionID != "" {
		version, ok := fa.versions[versionID]
		if !ok {
			return nil, fmt.Errorf("profile version %s not found", versionID)
		}
		if version.DeviceTypeID != deviceTypeID {
			return nil, fmt.Errorf("profile_version_id %s does not belong to profile_id %s", versionID, deviceTypeID)
		}
		// Deliberately no status check: this is what Atom does.
		return version, nil
	}

	versions := fa.versionsOf(deviceTypeID)
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].Status == DeviceTypeVersionStatusActive {
			return versions[i], nil
		}
	}
	return nil, fmt.Errorf("profile %s has no active version", deviceTypeID)
}

func (fa *fakeAtomDeviceTypes) createEntity(w http.ResponseWriter, payload gqlPayload) {
	input, _ := payload.Variables["input"].(map[string]any)
	entity := &fakeEntity{
		ID:       stringField(input, "id"),
		Kind:     stringField(input, "kind"),
		Name:     stringField(input, "name"),
		TenantID: stringField(input, "tenantId"),
		Status:   atomStatusActive,
	}
	if entity.ID == "" {
		entity.ID = fa.id("device")
	}
	entity.Attributes, _ = input["attributes"].(map[string]any)
	if entity.Attributes == nil {
		entity.Attributes = map[string]any{}
	}

	if deviceTypeID := stringField(input, "profileId"); deviceTypeID != "" {
		version, err := fa.resolveBinding(deviceTypeID, stringField(input, "profileVersionId"))
		if err != nil {
			fa.fail(w, "%s", err.Error())
			return
		}
		if messages := validateAgainstSchema(version.JSONSchema, entity.Attributes); len(messages) > 0 {
			fa.fail(w, "%s%s", atomSchemaValidationPrefix, strings.Join(messages, "; "))
			return
		}
		entity.DeviceTypeID = deviceTypeID
		entity.DeviceTypeVersionID = version.ID
		entity.Kind = atomKindDevice
	}

	fa.entities[entity.ID] = entity
	_ = writeJSON(w, map[string]any{"data": map[string]any{"createEntity": deviceTypeEntityJSON(entity)}})
}

func (fa *fakeAtomDeviceTypes) updateEntity(w http.ResponseWriter, payload gqlPayload) {
	id, _ := payload.Variables["id"].(string)
	entity, ok := fa.entities[id]
	if !ok {
		fa.fail(w, "entity %s not found", id)
		return
	}
	input, _ := payload.Variables["input"].(map[string]any)

	// Atom validates against the version already bound, looked up by the
	// entity's stored profile_version_id — not against a binding supplied in
	// the same call.
	if attributes, ok := input["attributes"].(map[string]any); ok {
		if version, bound := fa.versions[entity.DeviceTypeVersionID]; bound {
			if messages := validateAgainstSchema(version.JSONSchema, attributes); len(messages) > 0 {
				fa.fail(w, "%s%s", atomSchemaValidationPrefix, strings.Join(messages, "; "))
				return
			}
		}
		entity.Attributes = attributes
	}
	if name := stringField(input, "name"); name != "" {
		entity.Name = name
	}
	if status := stringField(input, "status"); status != "" {
		entity.Status = status
	}
	// COALESCE: an absent binding keeps the stored one.
	if deviceTypeID := stringField(input, "profileId"); deviceTypeID != "" {
		version, err := fa.resolveBinding(deviceTypeID, stringField(input, "profileVersionId"))
		if err != nil {
			fa.fail(w, "%s", err.Error())
			return
		}
		entity.DeviceTypeID = deviceTypeID
		entity.DeviceTypeVersionID = version.ID
	}

	_ = writeJSON(w, map[string]any{"data": map[string]any{"updateEntity": deviceTypeEntityJSON(entity)}})
}

func (fa *fakeAtomDeviceTypes) getEntity(w http.ResponseWriter, payload gqlPayload) {
	id, _ := payload.Variables["id"].(string)
	entity, ok := fa.entities[id]
	if !ok {
		fa.fail(w, "entity %s not found", id)
		return
	}
	_ = writeJSON(w, map[string]any{"data": map[string]any{"entity": deviceTypeEntityJSON(entity)}})
}

func (fa *fakeAtomDeviceTypes) listEntities(w http.ResponseWriter, payload gqlPayload) {
	deviceTypeID, _ := payload.Variables["profileId"].(string)
	kind, _ := payload.Variables["kind"].(string)

	var items []map[string]any
	for _, entity := range fa.entities {
		if deviceTypeID != "" && entity.DeviceTypeID != deviceTypeID {
			continue
		}
		if kind != "" && entity.Kind != kind {
			continue
		}
		items = append(items, deviceTypeEntityJSON(entity))
	}
	sort.Slice(items, func(i, j int) bool { return items[i]["id"].(string) < items[j]["id"].(string) })

	_ = writeJSON(w, map[string]any{
		"data": map[string]any{"entities": map[string]any{"items": items, "total": len(items)}},
	})
}

func deviceTypeJSON(deviceType *fakeDeviceType) map[string]any {
	return map[string]any{
		"id":          deviceType.ID,
		"tenant_id":   deviceType.TenantID,
		"key":         deviceType.Key,
		"name":        deviceType.Name,
		"description": deviceType.Description,
		"status":      deviceType.Status,
	}
}

func versionJSON(version *fakeDeviceTypeVersion) map[string]any {
	return map[string]any{
		"id":             version.ID,
		"device_type_id": version.DeviceTypeID,
		"version":        version.Version,
		"json_schema":    version.JSONSchema,
		"ui_schema":      version.UISchema,
		"status":         version.Status,
	}
}

func deviceTypeEntityJSON(entity *fakeEntity) map[string]any {
	return map[string]any{
		"id":                     entity.ID,
		"kind":                   entity.Kind,
		"name":                   entity.Name,
		"tenant_id":              entity.TenantID,
		"device_type_id":         entity.DeviceTypeID,
		"device_type_version_id": entity.DeviceTypeVersionID,
		"status":                 entity.Status,
		"attributes":             entity.Attributes,
	}
}

func stringField(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return value
}

// validateAgainstSchema is a small JSON Schema validator covering the keywords
// BuildCapabilitySchema emits, worded exactly as the jsonschema crate words
// them so the client's error translation is exercised against real input.
func validateAgainstSchema(schema, attributes map[string]any) []string {
	if len(schema) == 0 {
		return nil
	}
	var messages []string

	for _, name := range jsonSlice(schema["required"]) {
		name, ok := name.(string)
		if !ok {
			continue
		}
		if _, present := attributes[name]; !present {
			messages = append(messages, fmt.Sprintf("%q is a required property", name))
		}
	}

	properties, _ := schema["properties"].(map[string]any)
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		property, _ := properties[name].(map[string]any)
		value, present := attributes[name]
		if !present || property == nil {
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			continue
		}
		instance := string(encoded)

		declared, _ := property["type"].(string)
		if !matchesType(declared, value) {
			messages = append(messages, fmt.Sprintf("%s is not of type %q", instance, declared))
			continue
		}
		number, isNumber := floatValue(value)
		if isNumber {
			if min, ok := floatValue(property["minimum"]); ok && number < min {
				messages = append(messages, fmt.Sprintf("%s is less than the minimum of %s", instance, formatLimit(min)))
			}
			if max, ok := floatValue(property["maximum"]); ok && number > max {
				messages = append(messages, fmt.Sprintf("%s is greater than the maximum of %s", instance, formatLimit(max)))
			}
		}
		if options := jsonSlice(property["enum"]); len(options) > 0 {
			allowed := false
			for _, option := range options {
				if option == value {
					allowed = true
					break
				}
			}
			if !allowed {
				encodedOptions, _ := json.Marshal(options)
				messages = append(messages, fmt.Sprintf("%s is not one of %s", instance, encodedOptions))
			}
		}
	}

	return messages
}

func matchesType(declared string, value any) bool {
	switch declared {
	case ValueTypeNumber:
		_, ok := floatValue(value)
		return ok
	case ValueTypeInteger:
		number, ok := floatValue(value)
		return ok && number == float64(int64(number))
	case ValueTypeString:
		_, ok := value.(string)
		return ok
	case ValueTypeBoolean:
		_, ok := value.(bool)
		return ok
	default:
		return true
	}
}

func formatLimit(v float64) string {
	encoded, _ := json.Marshal(v)
	return string(encoded)
}

// watermeter is the running example: a type that reports volume and battery
// and accepts a reporting-interval command.
func watermeter() CapabilityDocument {
	return CapabilityDocument{
		Measurements: []Measurement{
			{Name: "volume", Unit: "m3", Access: MeasurementAccessRead, Required: true},
			{Name: "battery", Unit: "%", Access: MeasurementAccessRead, Min: floatPtr(0), Max: floatPtr(100)},
		},
		Commands: []Command{
			{Name: "set_interval", Params: map[string]string{"seconds": "int"}},
		},
	}
}

func createWatermeterType(t *testing.T, client *Client) (DeviceType, DeviceTypeVersion) {
	t.Helper()
	ctx := context.Background()

	deviceType, err := client.CreateDeviceType(ctx, DeviceType{
		TenantID:    testTenantID,
		Key:         "watermeter",
		Name:        "Watermeter",
		Description: "Volumetric water meter",
	})
	if err != nil {
		t.Fatalf("create device type: %v", err)
	}
	version, err := client.CreateDeviceTypeVersion(ctx, deviceType.ID, DeviceTypeVersion{Capabilities: watermeter()})
	if err != nil {
		t.Fatalf("create device type version: %v", err)
	}
	return deviceType, version
}

// Acceptance criterion 1.
func TestDeviceTypeCapabilityDocumentSurvivesAtom(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, created := createWatermeterType(t, client)
	if created.Version != 1 {
		t.Fatalf("first version must be 1, got %d", created.Version)
	}

	versions, err := client.ListDeviceTypeVersions(ctx, deviceType.ID)
	if err != nil {
		t.Fatalf("list device type versions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one version, got %d", len(versions))
	}

	got := versions[0].Capabilities
	if len(got.Measurements) != 2 || len(got.Commands) != 1 {
		t.Fatalf("declaration did not survive the round trip: %+v", got)
	}
	if got.Measurements[0].Name != "volume" || got.Measurements[0].Unit != "m3" {
		t.Fatalf("unexpected first measurement: %+v", got.Measurements[0])
	}
	if got.Commands[0].Params["seconds"] != ValueTypeInteger {
		t.Fatalf("unexpected command params: %+v", got.Commands[0].Params)
	}
}

// Acceptance criterion 2.
func TestDeviceBoundToDeviceTypeAcceptsConformingAttributes(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, version := createWatermeterType(t, client)

	device, err := client.CreateEntity(ctx, Entity{
		Kind:         atomKindDevice,
		Name:         "meter-1",
		TenantID:     testTenantID,
		DeviceTypeID: deviceType.ID,
		Attributes:   Attributes{"volume": 12.5, "battery": 90, "source": "magistrala"},
	})
	if err != nil {
		t.Fatalf("create bound device: %v", err)
	}
	if device.DeviceTypeID != deviceType.ID {
		t.Fatalf("device is not bound to the type: %+v", device)
	}
	if device.DeviceTypeVersionID != version.ID {
		t.Fatalf("expected a bind to the active version %s, got %s", version.ID, device.DeviceTypeVersionID)
	}
}

// Acceptance criterion 3: every violation shape names its field.
func TestDeviceBoundToDeviceTypeRejectsViolatingAttributes(t *testing.T) {
	cases := []struct {
		name       string
		attributes Attributes
		wantField  string
		constraint string
	}{
		{
			name:       "wrong type",
			attributes: Attributes{"volume": "quite a lot"},
			wantField:  "volume",
			constraint: "type",
		},
		{
			name:       "missing required",
			attributes: Attributes{"battery": 90},
			wantField:  "volume",
			constraint: "required",
		},
		{
			name:       "out of range",
			attributes: Attributes{"volume": 12.5, "battery": 140},
			wantField:  "battery",
			constraint: "maximum",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, client := newFakeAtomDeviceTypes(t)
			deviceType, _ := createWatermeterType(t, client)

			_, err := client.CreateEntity(context.Background(), Entity{
				Kind:         atomKindDevice,
				Name:         "meter-1",
				TenantID:     testTenantID,
				DeviceTypeID: deviceType.ID,
				Attributes:   tc.attributes,
			})
			if err == nil {
				t.Fatal("expected the write to be rejected")
			}

			schemaErr, ok := AsSchemaValidationError(err)
			if !ok {
				t.Fatalf("expected a typed schema error, got %T: %v", err, err)
			}
			if len(schemaErr.Violations) != 1 {
				t.Fatalf("expected one violation, got %+v", schemaErr.Violations)
			}
			if got := schemaErr.Violations[0].Field; got != tc.wantField {
				t.Fatalf("expected the error to name %q, got %q", tc.wantField, got)
			}
			if got := schemaErr.Violations[0].Constraint; got != tc.constraint {
				t.Fatalf("expected constraint %q, got %q", tc.constraint, got)
			}
			if !strings.Contains(err.Error(), tc.wantField) {
				t.Fatalf("error text must name the field, got %q", err.Error())
			}
		})
	}
}

// Acceptance criterion 4, as decided: Magistrala scopes device types to a
// tenant and does not expose Atom's global ones, because Atom's only
// alternative to "this tenant" is "every tenant".
func TestListDeviceTypesIsAlwaysScopedToATenant(t *testing.T) {
	fa, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	createWatermeterType(t, client)
	fa.deviceTypes["global"] = &fakeDeviceType{
		ID: "global", ObjectKind: atomObjectKindEntity, Kind: atomKindDevice,
		Key: "global-sensor", Name: "Global sensor", Status: DeviceTypeStatusActive,
	}
	fa.deviceTypes["other-tenant"] = &fakeDeviceType{
		ID: "other-tenant", TenantID: "tenant-2", ObjectKind: atomObjectKindEntity, Kind: atomKindDevice,
		Key: "someone-elses", Name: "Someone else's", Status: DeviceTypeStatusActive,
	}

	list, err := client.ListDeviceTypes(ctx, DeviceTypeQuery{TenantID: testTenantID})
	if err != nil {
		t.Fatalf("list device types: %v", err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Key != "watermeter" {
		t.Fatalf("expected only the tenant's own device types, got: %+v", list)
	}

	if len(fa.listQueries) != 1 {
		t.Fatalf("expected one list query, got %d", len(fa.listQueries))
	}
	query := fa.listQueries[0]
	if query["tenantId"] != testTenantID {
		t.Fatalf("every listing must carry a tenant, got: %+v", query)
	}
	if query["objectKind"] != atomObjectKindEntity || query["kind"] != atomKindDevice {
		t.Fatalf("listing must be narrowed to device types, got: %+v", query)
	}

	if _, err := client.ListDeviceTypes(ctx, DeviceTypeQuery{}); err == nil {
		t.Fatal("expected a tenant-less listing to be refused")
	}
}

// Acceptance criterion 5.
func TestListEntitiesByDeviceTypeReturnsExactlyTheBoundDevices(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	watermeterType, _ := createWatermeterType(t, client)

	probeType, err := client.CreateDeviceType(ctx, DeviceType{
		TenantID: testTenantID, Key: "probe", Name: "Temperature probe",
	})
	if err != nil {
		t.Fatalf("create device type: %v", err)
	}
	if _, err := client.CreateDeviceTypeVersion(ctx, probeType.ID, DeviceTypeVersion{
		Capabilities: CapabilityDocument{Measurements: []Measurement{{Name: "temperature", Unit: "Cel"}}},
	}); err != nil {
		t.Fatalf("create device type version: %v", err)
	}

	bound := map[string]bool{}
	for _, name := range []string{"meter-1", "meter-2"} {
		device, err := client.CreateEntity(ctx, Entity{
			Kind: atomKindDevice, Name: name, TenantID: testTenantID,
			DeviceTypeID: watermeterType.ID,
			Attributes:   Attributes{"volume": 1.0},
		})
		if err != nil {
			t.Fatalf("create bound device: %v", err)
		}
		bound[device.ID] = true
	}
	if _, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "probe-1", TenantID: testTenantID,
		DeviceTypeID: probeType.ID, Attributes: Attributes{"temperature": 21.0},
	}); err != nil {
		t.Fatalf("create bound device: %v", err)
	}
	if _, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "untyped-1", TenantID: testTenantID,
	}); err != nil {
		t.Fatalf("create untyped device: %v", err)
	}

	list, err := client.ListEntities(ctx, Query{DeviceTypeID: watermeterType.ID})
	if err != nil {
		t.Fatalf("list entities by device type: %v", err)
	}
	if len(list.Items) != 2 {
		t.Fatalf("expected exactly the two bound devices, got: %+v", list.Items)
	}
	for _, item := range list.Items {
		if !bound[item.ID] {
			t.Fatalf("unexpected device in the listing: %+v", item)
		}
	}
}

// Acceptance criterion 6.
func TestNewVersionLeavesDevicesOnTheOldVersionWorking(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, first := createWatermeterType(t, client)

	device, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-1", TenantID: testTenantID,
		DeviceTypeID: deviceType.ID,
		Attributes:   Attributes{"volume": 12.5},
	})
	if err != nil {
		t.Fatalf("create bound device: %v", err)
	}

	// Version 2 tightens the declaration: serial becomes mandatory.
	second, err := client.CreateDeviceTypeVersion(ctx, deviceType.ID, DeviceTypeVersion{
		Capabilities: CapabilityDocument{Measurements: []Measurement{
			{Name: "volume", Unit: "m3", Required: true},
			{Name: "serial", Type: ValueTypeString, Required: true},
		}},
	})
	if err != nil {
		t.Fatalf("create device type version: %v", err)
	}
	if second.Version != 2 {
		t.Fatalf("expected version 2, got %d", second.Version)
	}

	// The deployed device is still on version 1 and still writable without the
	// field version 2 added.
	updated, err := client.UpdateEntity(ctx, device.ID, Entity{Attributes: Attributes{"volume": 13.5}})
	if err != nil {
		t.Fatalf("device on version 1 must keep working: %v", err)
	}
	if updated.DeviceTypeVersionID != first.ID {
		t.Fatalf("device must stay on version 1, got %s", updated.DeviceTypeVersionID)
	}

	// New devices land on version 2, which does enforce the new field.
	if _, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-2", TenantID: testTenantID,
		DeviceTypeID: deviceType.ID, Attributes: Attributes{"volume": 1.0},
	}); err == nil {
		t.Fatal("expected version 2 to reject a device missing its new required field")
	}
}

// Acceptance criterion 7, at the device type level — the only deprecation Atom
// enforces server-side.
func TestDeprecatedDeviceTypeBlocksNewBindingsAndLeavesExistingIntact(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, _ := createWatermeterType(t, client)
	device, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-1", TenantID: testTenantID,
		DeviceTypeID: deviceType.ID, Attributes: Attributes{"volume": 12.5},
	})
	if err != nil {
		t.Fatalf("create bound device: %v", err)
	}

	if _, err := client.UpdateDeviceType(ctx, deviceType.ID, DeviceType{Status: DeviceTypeStatusDeprecated}); err != nil {
		t.Fatalf("deprecate device type: %v", err)
	}

	if _, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-2", TenantID: testTenantID,
		DeviceTypeID: deviceType.ID, Attributes: Attributes{"volume": 1.0},
	}); err == nil {
		t.Fatal("expected a deprecated device type to refuse new bindings")
	}

	existing, err := client.GetEntity(ctx, device.ID)
	if err != nil {
		t.Fatalf("existing device must remain readable: %v", err)
	}
	if existing.DeviceTypeID != deviceType.ID {
		t.Fatalf("existing binding must be untouched, got: %+v", existing)
	}
	if _, err := client.UpdateEntity(ctx, device.ID, Entity{Attributes: Attributes{"volume": 14.0}}); err != nil {
		t.Fatalf("existing device must remain writable: %v", err)
	}
}

// Acceptance criterion 7, at the version level. Atom binds a caller-named
// version whatever its status, so without this check a non-active version
// would still accept new devices.
func TestBindDeviceTypeRefusesNonActiveVersion(t *testing.T) {
	fa, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, first := createWatermeterType(t, client)
	staged, err := client.CreateDeviceTypeVersion(ctx, deviceType.ID, DeviceTypeVersion{
		Status:       DeviceTypeVersionStatusDeprecated,
		Capabilities: watermeter(),
	})
	if err != nil {
		t.Fatalf("create device type version: %v", err)
	}

	device, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-1", TenantID: testTenantID,
		Attributes: Attributes{"volume": 12.5},
	})
	if err != nil {
		t.Fatalf("create device: %v", err)
	}

	if _, err := client.BindDeviceType(ctx, device.ID, deviceType.ID, staged.ID); !errors.Is(err, ErrDeviceTypeVersionNotBindable) {
		t.Fatalf("expected ErrDeviceTypeVersionNotBindable, got %v", err)
	}
	if fa.entities[device.ID].DeviceTypeID != "" {
		t.Fatalf("a refused bind must not write: %+v", fa.entities[device.ID])
	}

	bound, err := client.BindDeviceType(ctx, device.ID, deviceType.ID, first.ID)
	if err != nil {
		t.Fatalf("binding to the active version must succeed: %v", err)
	}
	if bound.DeviceTypeVersionID != first.ID {
		t.Fatalf("expected a bind to version 1, got %s", bound.DeviceTypeVersionID)
	}
}

func TestBindDeviceTypeRefusesNonActiveDeviceType(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, version := createWatermeterType(t, client)
	device, err := client.CreateEntity(ctx, Entity{Kind: atomKindDevice, Name: "meter-1", TenantID: testTenantID})
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	if _, err := client.UpdateDeviceType(ctx, deviceType.ID, DeviceType{Status: DeviceTypeStatusDisabled}); err != nil {
		t.Fatalf("disable device type: %v", err)
	}

	if _, err := client.BindDeviceType(ctx, device.ID, deviceType.ID, version.ID); !errors.Is(err, ErrDeviceTypeNotBindable) {
		t.Fatalf("expected ErrDeviceTypeNotBindable, got %v", err)
	}
}

func TestActiveDeviceTypeVersionPicksTheHighestActive(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, first := createWatermeterType(t, client)

	// A staged draft must not become the binding target.
	if _, err := client.CreateDeviceTypeVersion(ctx, deviceType.ID, DeviceTypeVersion{
		Status: DeviceTypeVersionStatusDraft, Capabilities: watermeter(),
	}); err != nil {
		t.Fatalf("create draft version: %v", err)
	}

	active, err := client.ActiveDeviceTypeVersion(ctx, deviceType.ID)
	if err != nil {
		t.Fatalf("active device type version: %v", err)
	}
	if active.ID != first.ID {
		t.Fatalf("expected version 1 to remain active, got %+v", active)
	}

	third, err := client.CreateDeviceTypeVersion(ctx, deviceType.ID, DeviceTypeVersion{Capabilities: watermeter()})
	if err != nil {
		t.Fatalf("create device type version: %v", err)
	}
	if third.Version != 3 {
		t.Fatalf("expected version 3, got %d", third.Version)
	}

	active, err = client.ActiveDeviceTypeVersion(ctx, deviceType.ID)
	if err != nil {
		t.Fatalf("active device type version: %v", err)
	}
	if active.ID != third.ID {
		t.Fatalf("expected the newest active version, got %+v", active)
	}
}

func TestCreateDeviceTypeRequiresTenantAndKey(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	if _, err := client.CreateDeviceType(ctx, DeviceType{Key: "watermeter", Name: "Watermeter"}); err == nil {
		t.Fatal("expected a tenant-less device type to be refused")
	}
	if _, err := client.CreateDeviceType(ctx, DeviceType{TenantID: testTenantID, Name: "Watermeter"}); err == nil {
		t.Fatal("expected a keyless device type to be refused")
	}
}

// Binding is schema validation, not default-value injection: it must not
// invent attributes the caller did not write.
func TestBindingDoesNotPopulateAttributes(t *testing.T) {
	_, client := newFakeAtomDeviceTypes(t)
	ctx := context.Background()

	deviceType, _ := createWatermeterType(t, client)
	device, err := client.CreateEntity(ctx, Entity{
		Kind: atomKindDevice, Name: "meter-1", TenantID: testTenantID,
		DeviceTypeID: deviceType.ID,
		Attributes:   Attributes{"volume": 12.5},
	})
	if err != nil {
		t.Fatalf("create bound device: %v", err)
	}
	if _, invented := device.Attributes["battery"]; invented {
		t.Fatalf("binding must not populate declared-but-unwritten attributes: %+v", device.Attributes)
	}
	if len(device.Attributes) != 1 {
		t.Fatalf("expected only the attributes written, got: %+v", device.Attributes)
	}
}
