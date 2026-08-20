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
	"time"
)

const testTimeout = 5 * time.Second

func TestGraphQLClientAddsBearerTokenAndDecodesData(t *testing.T) {
	// The handler runs on its own goroutine, so it only records what it saw and
	// the assertions happen back on the test goroutine.
	var (
		method string
		authz  string
		req    graphQLRequest
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		authz = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tenant":{"id":"domain-1","name":"Domain"}}}`))
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "test-token", testTimeout)
	value, err := client.do(context.Background(), "query { tenant { id } }", map[string]any{varID: "domain-1"}, respTenant)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if got, want := method, http.MethodPost; got != want {
		t.Errorf("unexpected method: got %s want %s", got, want)
	}
	if got, want := authz, "Bearer test-token"; got != want {
		t.Errorf("unexpected authorization header: got %q want %q", got, want)
	}
	if !strings.Contains(req.Query, respTenant) {
		t.Errorf("query was not sent: %q", req.Query)
	}
	if got, want := req.Variables[varID], "domain-1"; got != want {
		t.Errorf("unexpected variable: got %v want %v", got, want)
	}

	var tenant map[string]any
	if err := json.Unmarshal(value, &tenant); err != nil {
		t.Fatalf("failed to decode tenant: %v", err)
	}
	if got, want := tenant[varID], "domain-1"; got != want {
		t.Errorf("unexpected tenant id: got %v want %v", got, want)
	}
}

func TestGraphQLClientTrimsEndpointAndToken(t *testing.T) {
	var authz string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tenant":{"id":"domain-1"}}}`))
	}))
	defer server.Close()

	client := newGraphQLClient(" "+server.URL+"\n", " test-token\n", testTimeout)
	if _, err := client.do(context.Background(), "query { tenant { id } }", nil, respTenant); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if got, want := authz, "Bearer test-token"; got != want {
		t.Errorf("unexpected authorization header: got %q want %q", got, want)
	}
}

func TestGraphQLClientReportsNullField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"tenant":null}}`))
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "", testTimeout)
	_, err := client.do(context.Background(), "query { tenant { id } }", nil, respTenant)
	if err == nil {
		t.Fatal("expected a not found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLClientReturnsGraphQLErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors":[{"message":"forbidden"},{"message":"invalid input"}]}`))
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "", testTimeout)
	_, err := client.do(context.Background(), "query { tenants { total } }", nil, respTenants)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "forbidden; invalid input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLClientReturnsHTTPStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	client := newGraphQLClient(server.URL, "", testTimeout)
	_, err := client.do(context.Background(), "query { tenants { total } }", nil, respTenants)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLClientRequiresEndpoint(t *testing.T) {
	client := newGraphQLClient("   ", "", testTimeout)
	if _, err := client.do(context.Background(), "query { tenants { total } }", nil, respTenants); err == nil {
		t.Fatal("expected endpoint error")
	}
}
