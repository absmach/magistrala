// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala/cli"
	"github.com/absmach/magistrala/pkg/atom"
)

type gqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

// fakeAtom is a minimal in-memory Atom GraphQL stand-in for command-level CLI
// tests, mirroring pkg/atom/devices_test.go's fake server on the atom side —
// duplicated rather than imported since that helper is unexported there.
type fakeAtom struct {
	t        *testing.T
	entities map[string]atom.Entity
	seq      int
	requests []gqlRequest

	getCount    map[string]int
	mutateAtGet map[string]int
	mutateAttrs map[string]atom.Attributes

	deviceTypes        map[string]atom.DeviceType
	typeVersions       map[string][]atom.DeviceTypeVersion
	typeSeq            int
	versionSeq         int
	updateEntityGQLErr string
}

// mutateOnNthGet swaps id's attributes in after its nth GetEntity call,
// simulating a concurrent write landing between the CLI's DeviceGateways
// read and SetDeviceGateways' own internal re-read.
func (fa *fakeAtom) mutateOnNthGet(id string, n int, attrs atom.Attributes) {
	if fa.mutateAtGet == nil {
		fa.mutateAtGet = map[string]int{}
		fa.mutateAttrs = map[string]atom.Attributes{}
	}
	fa.mutateAtGet[id] = n
	fa.mutateAttrs[id] = attrs
}

// seedDeviceType registers a device type, and seedDeviceTypeVersion one of its
// versions. They are methods rather than newFakeAtom arguments so the entity
// seeding signature the device and gateway tests use stays unchanged.
func (fa *fakeAtom) seedDeviceType(deviceTypes ...atom.DeviceType) {
	for _, dt := range deviceTypes {
		fa.deviceTypes[dt.ID] = dt
	}
}

func (fa *fakeAtom) seedDeviceTypeVersion(versions ...atom.DeviceTypeVersion) {
	for _, v := range versions {
		fa.typeVersions[v.DeviceTypeID] = append(fa.typeVersions[v.DeviceTypeID], v)
	}
}

// failEntityUpdate makes the next UpdateEntity answer with a GraphQL error,
// which is how Atom reports a device type schema rejection.
func (fa *fakeAtom) failEntityUpdate(message string) {
	fa.updateEntityGQLErr = message
}

func newFakeAtom(t *testing.T, seed ...atom.Entity) *fakeAtom {
	t.Helper()
	fa := &fakeAtom{
		t:            t,
		entities:     map[string]atom.Entity{},
		getCount:     map[string]int{},
		deviceTypes:  map[string]atom.DeviceType{},
		typeVersions: map[string][]atom.DeviceTypeVersion{},
	}
	for _, e := range seed {
		fa.entities[e.ID] = e
	}
	srv := httptest.NewServer(http.HandlerFunc(fa.handle))
	t.Cleanup(srv.Close)
	cli.SetAtomClient(atom.NewClient(atom.Config{URL: srv.URL}))
	return fa
}

func (fa *fakeAtom) handle(w http.ResponseWriter, r *http.Request) {
	var req gqlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fa.t.Fatalf("decode graphql request: %v", err)
	}
	fa.requests = append(fa.requests, req)
	w.Header().Set("Content-Type", "application/json")

	// CreateDeviceTypeVersion is matched before CreateDeviceType, whose name is
	// a prefix of it.
	switch {
	case strings.Contains(req.Query, "mutation CreateEntity"):
		fa.handleCreate(w, req)
	case strings.Contains(req.Query, "mutation UpdateEntity"):
		fa.handleUpdate(w, req)
	case strings.Contains(req.Query, "mutation DeleteEntity"):
		fa.handleDelete(w, req)
	case strings.Contains(req.Query, "query Entity("):
		fa.handleGet(w, req)
	case strings.Contains(req.Query, "query EntityExistence("):
		fa.handleEntityExistence(w, req)
	case strings.Contains(req.Query, "query Entities("):
		fa.handleList(w, req)
	case strings.Contains(req.Query, "mutation CreateDeviceTypeVersion"):
		fa.handleCreateDeviceTypeVersion(w, req)
	case strings.Contains(req.Query, "mutation CreateDeviceType"):
		fa.handleCreateDeviceType(w, req)
	case strings.Contains(req.Query, "mutation UpdateDeviceType"):
		fa.handleUpdateDeviceType(w, req)
	case strings.Contains(req.Query, "query DeviceTypeVersions("):
		fa.handleListDeviceTypeVersions(w, req)
	case strings.Contains(req.Query, "query DeviceTypes("):
		fa.handleListDeviceTypes(w, req)
	case strings.Contains(req.Query, "query DeviceType("):
		fa.handleGetDeviceType(w, req)
	default:
		fa.t.Fatalf("unexpected graphql query: %s", req.Query)
	}
}

func (fa *fakeAtom) handleCreateDeviceType(w http.ResponseWriter, req gqlRequest) {
	input, _ := req.Variables["input"].(map[string]any)
	fa.typeSeq++
	dt := atom.DeviceType{
		ID:          fmt.Sprintf("device-type-%d", fa.typeSeq),
		TenantID:    strVal(input["tenantId"]),
		Key:         strVal(input["key"]),
		Name:        strVal(input["displayName"]),
		Description: strVal(input["description"]),
		Status:      "active",
	}
	if status := strVal(input["status"]); status != "" {
		dt.Status = status
	}
	fa.deviceTypes[dt.ID] = dt
	writeGQLData(fa.t, w, map[string]any{"createProfile": deviceTypeJSON(dt)})
}

func (fa *fakeAtom) handleGetDeviceType(w http.ResponseWriter, req gqlRequest) {
	id, _ := req.Variables["id"].(string)
	dt, ok := fa.deviceTypes[id]
	if !ok {
		writeGQLError(fa.t, w, "device type not found")
		return
	}
	writeGQLData(fa.t, w, map[string]any{"profile": deviceTypeJSON(dt)})
}

func (fa *fakeAtom) handleListDeviceTypes(w http.ResponseWriter, req gqlRequest) {
	tenantID := strVal(req.Variables["tenantId"])
	status := strVal(req.Variables["status"])

	items := []map[string]any{}
	for _, dt := range fa.deviceTypes {
		if tenantID != "" && dt.TenantID != tenantID {
			continue
		}
		if status != "" && dt.Status != status {
			continue
		}
		items = append(items, deviceTypeJSON(dt))
	}
	writeGQLData(fa.t, w, map[string]any{"profiles": map[string]any{"items": items, "total": len(items)}})
}

func (fa *fakeAtom) handleUpdateDeviceType(w http.ResponseWriter, req gqlRequest) {
	id, _ := req.Variables["id"].(string)
	dt, ok := fa.deviceTypes[id]
	if !ok {
		writeGQLError(fa.t, w, "device type not found")
		return
	}
	input, _ := req.Variables["input"].(map[string]any)
	if name := strVal(input["displayName"]); name != "" {
		dt.Name = name
	}
	if description := strVal(input["description"]); description != "" {
		dt.Description = description
	}
	if status := strVal(input["status"]); status != "" {
		dt.Status = status
	}
	fa.deviceTypes[id] = dt
	writeGQLData(fa.t, w, map[string]any{"updateProfile": deviceTypeJSON(dt)})
}

func (fa *fakeAtom) handleCreateDeviceTypeVersion(w http.ResponseWriter, req gqlRequest) {
	deviceTypeID := strVal(req.Variables["profileId"])
	input, _ := req.Variables["input"].(map[string]any)
	fa.versionSeq++

	version := atom.DeviceTypeVersion{
		ID:           fmt.Sprintf("device-type-version-%d", fa.versionSeq),
		DeviceTypeID: deviceTypeID,
		Version:      intVal(input["version"]),
		Status:       "active",
		JSONSchema:   mapVal(input["jsonSchema"]),
		UISchema:     mapVal(input["uiSchema"]),
	}
	if status := strVal(input["status"]); status != "" {
		version.Status = status
	}
	fa.typeVersions[deviceTypeID] = append(fa.typeVersions[deviceTypeID], version)
	writeGQLData(fa.t, w, map[string]any{"createProfileVersion": deviceTypeVersionJSON(version)})
}

func (fa *fakeAtom) handleListDeviceTypeVersions(w http.ResponseWriter, req gqlRequest) {
	deviceTypeID := strVal(req.Variables["profileId"])
	items := []map[string]any{}
	for _, version := range fa.typeVersions[deviceTypeID] {
		items = append(items, deviceTypeVersionJSON(version))
	}
	writeGQLData(fa.t, w, map[string]any{"profileVersions": items})
}

func deviceTypeJSON(dt atom.DeviceType) map[string]any {
	return map[string]any{
		"id":          dt.ID,
		"tenant_id":   dt.TenantID,
		"key":         dt.Key,
		"name":        dt.Name,
		"description": dt.Description,
		"status":      dt.Status,
	}
}

func deviceTypeVersionJSON(v atom.DeviceTypeVersion) map[string]any {
	return map[string]any{
		"id":             v.ID,
		"device_type_id": v.DeviceTypeID,
		"version":        v.Version,
		"json_schema":    v.JSONSchema,
		"ui_schema":      v.UISchema,
		"status":         v.Status,
	}
}

func intVal(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func mapVal(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func (fa *fakeAtom) handleCreate(w http.ResponseWriter, req gqlRequest) {
	input, _ := req.Variables["input"].(map[string]any)
	fa.seq++
	e := atom.Entity{
		ID:         fmt.Sprintf("device-%d", fa.seq),
		Kind:       strVal(input["kind"]),
		Name:       strVal(input["name"]),
		TenantID:   strVal(input["tenantId"]),
		Status:     "active",
		Attributes: attrsVal(input["attributes"]),
	}
	fa.entities[e.ID] = e
	writeGQLData(fa.t, w, map[string]any{"createEntity": entityJSON(e)})
}

func (fa *fakeAtom) handleUpdate(w http.ResponseWriter, req gqlRequest) {
	if message := fa.updateEntityGQLErr; message != "" {
		fa.updateEntityGQLErr = ""
		writeGQLError(fa.t, w, message)
		return
	}

	id, _ := req.Variables["id"].(string)
	e, ok := fa.entities[id]
	if !ok {
		writeGQLError(fa.t, w, "entity not found")
		return
	}
	input, _ := req.Variables["input"].(map[string]any)
	if name := strVal(input["name"]); name != "" {
		e.Name = name
	}
	if status := strVal(input["status"]); status != "" {
		e.Status = status
	}
	if profileID := strVal(input["profileId"]); profileID != "" {
		e.DeviceTypeID = profileID
	}
	if versionID := strVal(input["profileVersionId"]); versionID != "" {
		e.DeviceTypeVersionID = versionID
	}
	if attrs, ok := input["attributes"].(map[string]any); ok {
		e.Attributes = atom.Attributes(attrs)
	}
	fa.entities[id] = e
	writeGQLData(fa.t, w, map[string]any{"updateEntity": entityJSON(e)})
}

func (fa *fakeAtom) handleDelete(w http.ResponseWriter, req gqlRequest) {
	id, _ := req.Variables["id"].(string)
	if _, ok := fa.entities[id]; !ok {
		writeGQLError(fa.t, w, "entity not found")
		return
	}
	delete(fa.entities, id)
	writeGQLData(fa.t, w, map[string]any{"deleteEntity": true})
}

func (fa *fakeAtom) handleGet(w http.ResponseWriter, req gqlRequest) {
	id, _ := req.Variables["id"].(string)
	fa.getCount[id]++
	if n, ok := fa.mutateAtGet[id]; ok && fa.getCount[id] == n {
		e := fa.entities[id]
		e.Attributes = fa.mutateAttrs[id]
		fa.entities[id] = e
	}

	e, ok := fa.entities[id]
	if !ok {
		writeGQLError(fa.t, w, "entity not found")
		return
	}
	writeGQLData(fa.t, w, map[string]any{"entity": entityJSON(e)})
}

// handleEntityExistence answers the batched aliased-entity existence check
// pkg/atom's liveGateways uses (see entitiesExist/entityBatch), one round
// trip for however many ids it was asked about rather than one per id.
func (fa *fakeAtom) handleEntityExistence(w http.ResponseWriter, req gqlRequest) {
	data := map[string]any{}
	for i := 0; ; i++ {
		raw, ok := req.Variables[fmt.Sprintf("id%d", i)]
		if !ok {
			break
		}
		id, _ := raw.(string)
		alias := fmt.Sprintf("e%d", i)
		if _, exists := fa.entities[id]; exists {
			data[alias] = map[string]any{"id": id}
		} else {
			data[alias] = nil
		}
	}
	writeGQLData(fa.t, w, data)
}

func (fa *fakeAtom) handleList(w http.ResponseWriter, req gqlRequest) {
	kind := strVal(req.Variables["kind"])
	tenantID := strVal(req.Variables["tenantId"])
	contains, _ := req.Variables["attributesContains"].(map[string]any)

	var items []map[string]any
	for _, e := range fa.entities {
		if kind != "" && e.Kind != kind {
			continue
		}
		if tenantID != "" && e.TenantID != tenantID {
			continue
		}
		if contains != nil && !entityMatchesContains(e, contains) {
			continue
		}
		items = append(items, entityJSON(e))
	}
	writeGQLData(fa.t, w, map[string]any{"entities": map[string]any{"items": items, "total": len(items)}})
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

func attrsVal(v any) atom.Attributes {
	m, _ := v.(map[string]any)
	if m == nil {
		return atom.Attributes{}
	}
	return atom.Attributes(m)
}

func entityJSON(e atom.Entity) map[string]any {
	attrs := map[string]any(e.Attributes)
	if attrs == nil {
		attrs = map[string]any{}
	}
	return map[string]any{
		"id":                     e.ID,
		"kind":                   e.Kind,
		"name":                   e.Name,
		"tenant_id":              e.TenantID,
		"device_type_id":         e.DeviceTypeID,
		"device_type_version_id": e.DeviceTypeVersionID,
		"status":                 e.Status,
		"attributes":             attrs,
	}
}

// entityMatchesContains reimplements Atom's JSONB containment just enough
// for these tests: every key must be present, list values match by element.
// Attributes seeded directly hold []string; attributes that passed through a
// real GraphQL round-trip decode to []any — normalise both before comparing.
func entityMatchesContains(e atom.Entity, contains map[string]any) bool {
	for k, want := range contains {
		got, ok := e.Attributes[k]
		if !ok {
			return false
		}
		wantList := toStringSlice(want)
		if wantList == nil {
			if got != want {
				return false
			}
			continue
		}
		gotList := toStringSlice(got)
		for _, w := range wantList {
			found := false
			for _, g := range gotList {
				if g == w {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func toStringSlice(v any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func writeGQLData(t *testing.T, w http.ResponseWriter, data map[string]any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{"data": data}); err != nil {
		t.Fatalf("encode graphql response: %v", err)
	}
}

func writeGQLError(t *testing.T, w http.ResponseWriter, message string) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"errors": []map[string]string{{"message": message}},
	}); err != nil {
		t.Fatalf("encode graphql error response: %v", err)
	}
}
