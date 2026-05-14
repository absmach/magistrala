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
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
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
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var payload struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Variables["kind"] != KindRule || payload.Variables["tenantId"] != "domain-1" {
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
	got, err := client.ListResources(context.Background(), Query{Kind: KindRule, TenantID: "domain-1"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if got.Total != 1 || got.Items[0].ID != "rule-1" {
		t.Fatalf("unexpected list: %+v", got)
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
