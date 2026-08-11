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

func newFakeAtom(t *testing.T, seed ...atom.Entity) *fakeAtom {
	t.Helper()
	fa := &fakeAtom{t: t, entities: map[string]atom.Entity{}, getCount: map[string]int{}}
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

	switch {
	case strings.Contains(req.Query, "mutation CreateEntity"):
		fa.handleCreate(w, req)
	case strings.Contains(req.Query, "mutation UpdateEntity"):
		fa.handleUpdate(w, req)
	case strings.Contains(req.Query, "mutation DeleteEntity"):
		fa.handleDelete(w, req)
	case strings.Contains(req.Query, "query Entity("):
		fa.handleGet(w, req)
	case strings.Contains(req.Query, "query Entities("):
		fa.handleList(w, req)
	default:
		fa.t.Fatalf("unexpected graphql query: %s", req.Query)
	}
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
		"id":         e.ID,
		"kind":       e.Kind,
		"name":       e.Name,
		"tenant_id":  e.TenantID,
		"status":     e.Status,
		"attributes": attrs,
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
