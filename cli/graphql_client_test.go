// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGraphQLClientAddsBearerTokenAndDecodesData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("unexpected method: got %s want %s", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Fatalf("unexpected authorization header: got %q want %q", got, want)
		}
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if !strings.Contains(req.Query, "tenant") {
			t.Fatalf("query was not sent: %q", req.Query)
		}
		if got, want := req.Variables["id"], "domain-1"; got != want {
			t.Fatalf("unexpected variable: got %v want %v", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tenant":{"id":"domain-1","name":"Domain"}}}`))
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "test-token")
	var out struct {
		Tenant map[string]any `json:"tenant"`
	}
	if err := client.do(context.Background(), "query { tenant { id } }", map[string]any{"id": "domain-1"}, &out); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if got, want := out.Tenant["id"], "domain-1"; got != want {
		t.Fatalf("unexpected tenant id: got %v want %v", got, want)
	}
}

func TestGraphQLClientReturnsGraphQLErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"forbidden"},{"message":"invalid input"}]}`))
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "")
	err := client.do(context.Background(), "query { tenants { total } }", nil, &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "forbidden; invalid input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLClientReturnsHTTPStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "")
	err := client.do(context.Background(), "query { tenants { total } }", nil, &struct{}{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}
