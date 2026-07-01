// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpsertResourceCreatesThenUpdatesOnConflict(t *testing.T) {
	var operations []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		payload := string(body)
		switch {
		case strings.Contains(payload, "createResource"):
			operations = append(operations, "createResource")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"errors": []map[string]string{{"message": "duplicate key value violates unique constraint"}},
			})
			return
		case strings.Contains(payload, "updateResource"):
			operations = append(operations, "updateResource")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"updateResource": map[string]any{"id": "res-1", "kind": KindChannel, "name": "ch"},
				},
			})
			return
		default:
			t.Fatalf("unexpected GraphQL payload: %s", payload)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	if err := client.UpsertResource(context.Background(), Resource{ID: "res-1", Kind: KindChannel, Name: "ch"}); err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	want := []string{"createResource", "updateResource"}
	if len(operations) != len(want) {
		t.Fatalf("unexpected operation count: got %v want %v", operations, want)
	}
	for i := range want {
		if operations[i] != want[i] {
			t.Fatalf("unexpected operations: got %v want %v", operations, want)
		}
	}
}

func TestListResources(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Variables["kind"] != KindRule || payload.Variables["tenantId"] != testDomainID {
			t.Fatalf("unexpected variables: %+v", payload.Variables)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"resources": map[string]any{
					"items": []Resource{{ID: "rule-1", Kind: KindRule, Name: "high-temp"}},
					"total": 1,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.ListResources(context.Background(), Query{Kind: KindRule, TenantID: testDomainID})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if got.Total != 1 || got.Items[0].ID != "rule-1" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestCurrentAtomCompatibilitySurface(t *testing.T) {
	const (
		serviceToken = "service-token"
		runtimeToken = "runtime-token"
	)

	seen := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case atomGraphQLPath:
			if r.Method != http.MethodPost {
				t.Fatalf("unexpected GraphQL method: %s", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer "+serviceToken {
				t.Fatalf("unexpected GraphQL authorization header: %q", got)
			}
			var payload struct {
				Query     string         `json:"query"`
				Variables map[string]any `json:"variables"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			switch {
			case strings.Contains(payload.Query, "authzCheck"):
				seen["authzCheck"] = true
				input, ok := payload.Variables["input"].(map[string]any)
				if !ok {
					t.Fatalf("unexpected authz input: %+v", payload.Variables["input"])
				}
				if input["subjectId"] != testEntityID || input["action"] != atomActionPublish || input["resourceId"] != "channel-1" {
					t.Fatalf("unexpected authz input: %+v", input)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"authzCheck": map[string]any{"allowed": true, "reason": "allowed"},
					},
				})
			case strings.Contains(payload.Query, "authorizedObjectIds"):
				seen["authorizedObjectIds"] = true
				input, ok := payload.Variables["input"].(map[string]any)
				if !ok {
					t.Fatalf("unexpected authorized objects input: %+v", payload.Variables["input"])
				}
				if input["subjectId"] != testEntityID ||
					input["action"] != atomActionRead ||
					input["objectKind"] != atomObjectKindEntity ||
					input["objectType"] != atomKindDevice ||
					input["tenantId"] != testDomainID {
					t.Fatalf("unexpected authorized objects input: %+v", input)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"authorizedObjectIds": map[string]any{"ids": []string{testDeviceID}, "total": 1},
					},
				})
			case strings.Contains(payload.Query, "createApiKey"):
				seen["createApiKey"] = true
				if payload.Variables["entityId"] != testEntityID {
					t.Fatalf("unexpected createApiKey entity: %+v", payload.Variables)
				}
				input, ok := payload.Variables["input"].(map[string]any)
				if !ok || input["description"] != "Magistrala service token" {
					t.Fatalf("unexpected createApiKey input: %+v", payload.Variables["input"])
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{
						"createApiKey": map[string]any{
							"credentialId": testCredentialID,
							"key":          "atom_00000000000000000000000000000000_0000000000000000000000000000000000000000000000000000000000000000",
						},
					},
				})
			default:
				t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
			}
		case "/auth/introspect":
			seen["introspect"] = true
			if r.Method != http.MethodGet {
				t.Fatalf("unexpected introspection method: %s", r.Method)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer "+runtimeToken {
				t.Fatalf("unexpected introspection authorization header: %q", got)
			}
			_ = json.NewEncoder(w).Encode(IntrospectionResponse{
				Active:    true,
				EntityID:  testEntityID,
				TenantID:  testDomainID,
				SessionID: "session-1",
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Token: serviceToken, Timeout: time.Second})
	authz, err := client.CheckAuthz(context.Background(), AuthzRequest{
		SubjectID:  testEntityID,
		Action:     atomActionPublish,
		ResourceID: "channel-1",
	})
	if err != nil {
		t.Fatalf("authz check failed: %v", err)
	}
	if !authz.Allowed {
		t.Fatalf("unexpected authz response: %+v", authz)
	}

	objects, err := client.AuthorizedObjectIDs(context.Background(), AuthorizedObjectIDsQuery{
		SubjectID:  testEntityID,
		Action:     atomActionRead,
		ObjectKind: atomObjectKindEntity,
		ObjectType: atomKindDevice,
		TenantID:   testDomainID,
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("authorized object listing failed: %v", err)
	}
	if objects.Total != 1 || len(objects.IDs) != 1 || objects.IDs[0] != testDeviceID {
		t.Fatalf("unexpected authorized object listing: %+v", objects)
	}

	created, err := client.CreateAPIKey(context.Background(), testEntityID, "Magistrala service token")
	if err != nil {
		t.Fatalf("create API key failed: %v", err)
	}
	if created.CredentialID != testCredentialID || created.Key == "" {
		t.Fatalf("unexpected API key response: %+v", created)
	}

	introspection, err := client.Introspect(context.Background(), runtimeToken)
	if err != nil {
		t.Fatalf("introspection failed: %v", err)
	}
	if !introspection.Active || introspection.EntityID != testEntityID || introspection.TenantID != testDomainID {
		t.Fatalf("unexpected introspection response: %+v", introspection)
	}

	for _, operation := range []string{"authzCheck", "authorizedObjectIds", "createApiKey", "introspect"} {
		if !seen[operation] {
			t.Fatalf("operation %q was not exercised", operation)
		}
	}
}

func TestCreateTenantMapsRouteToAlias(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "createTenant") || !strings.Contains(payload.Query, "route: alias") {
			t.Fatalf("query does not map tenant alias to route: %s", payload.Query)
		}
		input, ok := payload.Variables["input"].(map[string]any)
		if !ok {
			t.Fatalf("unexpected input: %+v", payload.Variables["input"])
		}
		if input["alias"] != "d1" {
			t.Fatalf("expected alias input from route, got: %+v", input)
		}
		if _, ok := input["route"]; ok {
			t.Fatalf("input must not use Atom route field: %+v", input)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"createTenant": map[string]any{
					"id":     testTenantID,
					"name":   "D1",
					"route":  "d1",
					"status": "active",
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.CreateTenant(context.Background(), Tenant{Name: "D1", Route: "d1"})
	if err != nil {
		t.Fatalf("create tenant failed: %v", err)
	}
	if got.ID != testTenantID || got.Route != "d1" {
		t.Fatalf("unexpected tenant: %+v", got)
	}
}

func TestUpdateTenantMapsRouteToAlias(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "updateTenant") || !strings.Contains(payload.Query, "route: alias") {
			t.Fatalf("query does not map tenant alias to route: %s", payload.Query)
		}
		if payload.Variables["id"] != testTenantID {
			t.Fatalf("unexpected variables: %+v", payload.Variables)
		}
		input, ok := payload.Variables["input"].(map[string]any)
		if !ok {
			t.Fatalf("unexpected input: %+v", payload.Variables["input"])
		}
		if input["alias"] != "d1" {
			t.Fatalf("expected alias input from route, got: %+v", input)
		}
		if _, ok := input["route"]; ok {
			t.Fatalf("input must not use Atom route field: %+v", input)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"updateTenant": map[string]any{
					"id":     testTenantID,
					"name":   "D1",
					"route":  "d1",
					"status": "active",
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.UpdateTenant(context.Background(), testTenantID, Tenant{Name: "D1", Route: "d1"})
	if err != nil {
		t.Fatalf("update tenant failed: %v", err)
	}
	if got.ID != testTenantID || got.Route != "d1" {
		t.Fatalf("unexpected tenant: %+v", got)
	}
}

func TestListTenantsMapsRouteToAlias(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "$alias: String") ||
			!strings.Contains(payload.Query, "alias: $alias") ||
			!strings.Contains(payload.Query, "route: alias") {
			t.Fatalf("query does not use Atom alias for tenant route lookup: %s", payload.Query)
		}
		if strings.Contains(payload.Query, "$route") || strings.Contains(payload.Query, "route: $route") {
			t.Fatalf("query must not use removed Atom route field/filter: %s", payload.Query)
		}
		if payload.Variables["alias"] != "d1" {
			t.Fatalf("expected alias variable from route, got: %+v", payload.Variables)
		}
		if _, ok := payload.Variables["route"]; ok {
			t.Fatalf("variables must not use Atom route field: %+v", payload.Variables)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"tenants": map[string]any{
					"items": []Tenant{{ID: testTenantID, Name: "D1", Route: "d1", Status: "active"}},
					"total": 1,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.ListTenants(context.Background(), Query{Route: "d1", Limit: 1})
	if err != nil {
		t.Fatalf("list tenants failed: %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Route != "d1" {
		t.Fatalf("unexpected tenants: %+v", got)
	}
}

func TestCreateSharedKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "createSharedKey") {
			t.Fatalf("query does not create shared key: %s", payload.Query)
		}
		if payload.Variables["entityId"] != testDeviceID {
			t.Fatalf("unexpected entity id: %+v", payload.Variables)
		}
		input, ok := payload.Variables["input"].(map[string]any)
		if !ok {
			t.Fatalf("unexpected input: %+v", payload.Variables["input"])
		}
		if input["key"] != testClientKey || input["description"] != "provisioned from mg" {
			t.Fatalf("unexpected input: %+v", input)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"createSharedKey": map[string]any{
					"credentialId": testCredentialID,
					"key":          testClientKey,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.CreateSharedKey(context.Background(), testDeviceID, testClientKey, "provisioned from mg")
	if err != nil {
		t.Fatalf("create shared key failed: %v", err)
	}
	if got.CredentialID != testCredentialID || got.Key != testClientKey {
		t.Fatalf("unexpected shared key response: %+v", got)
	}
}

func TestRevealSharedKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "revealSharedKey") {
			t.Fatalf("query does not reveal shared key: %s", payload.Query)
		}
		if payload.Variables["entityId"] != testDeviceID || payload.Variables["credentialId"] != testCredentialID {
			t.Fatalf("unexpected variables: %+v", payload.Variables)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"revealSharedKey": map[string]any{
					"credentialId": testCredentialID,
					"key":          testClientKey,
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.RevealSharedKey(context.Background(), testDeviceID, testCredentialID)
	if err != nil {
		t.Fatalf("reveal shared key failed: %v", err)
	}
	if got.CredentialID != testCredentialID || got.Key != testClientKey {
		t.Fatalf("unexpected shared key response: %+v", got)
	}
}

func TestListCredentials(t *testing.T) {
	createdAt := "2026-06-30T10:15:30Z"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != atomGraphQLPath {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "credentials") {
			t.Fatalf("query does not list credentials: %s", payload.Query)
		}
		if payload.Variables["entityId"] != testDeviceID {
			t.Fatalf("unexpected variables: %+v", payload.Variables)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"credentials": map[string]any{
					"total": 1,
					"items": []map[string]any{{
						"id":         testCredentialID,
						"entity_id":  testDeviceID,
						"kind":       "shared_key",
						"identifier": "",
						"status":     "active",
						"created_at": createdAt,
					}},
				},
			},
		})
	}))
	defer srv.Close()

	client := NewClient(Config{URL: srv.URL, Timeout: time.Second})
	got, err := client.ListCredentials(context.Background(), testDeviceID)
	if err != nil {
		t.Fatalf("list credentials failed: %v", err)
	}
	if got.Total != 1 || len(got.Items) != 1 {
		t.Fatalf("unexpected credentials response: %+v", got)
	}
	item := got.Items[0]
	if item.ID != testCredentialID || item.EntityID != testDeviceID || item.Kind != "shared_key" || item.Status != "active" {
		t.Fatalf("unexpected credential item: %+v", item)
	}
	if item.CreatedAt.Format(time.RFC3339) != createdAt {
		t.Fatalf("unexpected created_at: %s", item.CreatedAt.Format(time.RFC3339))
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("ATOM_URL", "http://atom:8080/")
	t.Setenv("ATOM_ADMIN_TOKEN", "token")
	t.Setenv("ATOM_ADMIN_USERNAME", "admin")
	t.Setenv("ATOM_ADMIN_SECRET", "secret")
	t.Setenv("ATOM_TIMEOUT", "3s")

	cfg := LoadConfig()
	if cfg.URL != "http://atom:8080" || cfg.JWKSURL != "http://atom:8080/.well-known/jwks.json" || cfg.Token != "token" || cfg.AdminUsername != "admin" || cfg.AdminSecret != "secret" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.Timeout != 3*time.Second {
		t.Fatalf("unexpected timeout: %s", cfg.Timeout)
	}
}
