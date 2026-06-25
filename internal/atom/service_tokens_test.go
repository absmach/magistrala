// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProvisionServiceTokensCreatesMissingToken(t *testing.T) {
	fake := newFakeAtomTokenServer(t, nil)
	defer fake.Close()

	output := filepath.Join(t.TempDir(), ".env.tokens")
	result, err := ProvisionServiceTokens(context.Background(), fake.Client(), TokenProvisionOptions{
		OutputPath: output,
		Specs:      []ServiceTokenSpec{testTokenSpec()},
	})
	if err != nil {
		t.Fatalf("provision tokens failed: %v", err)
	}
	if !containsString(result.Created, testTokenSpec().Env) {
		t.Fatalf("expected token to be created, got result %+v", result)
	}
	if len(fake.created) != 1 {
		t.Fatalf("unexpected create count: %d", len(fake.created))
	}
	values, err := readTokenEnvFile(output)
	if err != nil {
		t.Fatalf("read token env file: %v", err)
	}
	if values[testTokenSpec().Env] == "" {
		t.Fatalf("expected generated token in env file")
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatalf("stat token env file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("unexpected token env permissions: got %s want -rw-------", got)
	}
	assertNoTempTokenFiles(t, filepath.Dir(output))
}

func TestProvisionServiceTokensPreservesExistingActiveToken(t *testing.T) {
	token := apiKeyForCredentialID("11111111-1111-1111-1111-111111111111")
	fake := newFakeAtomTokenServer(t, map[string]bool{token: true})
	defer fake.Close()

	output := filepath.Join(t.TempDir(), ".env.tokens")
	if err := os.WriteFile(output, []byte(testTokenSpec().Env+"="+token+"\n"), 0o600); err != nil {
		t.Fatalf("write existing token file: %v", err)
	}

	result, err := ProvisionServiceTokens(context.Background(), fake.Client(), TokenProvisionOptions{
		OutputPath: output,
		Specs:      []ServiceTokenSpec{testTokenSpec()},
	})
	if err != nil {
		t.Fatalf("provision tokens failed: %v", err)
	}
	if !containsString(result.Preserved, testTokenSpec().Env) {
		t.Fatalf("expected token to be preserved, got result %+v", result)
	}
	if len(fake.created) != 0 {
		t.Fatalf("expected no new API key, got %d", len(fake.created))
	}
	values, err := readTokenEnvFile(output)
	if err != nil {
		t.Fatalf("read token env file: %v", err)
	}
	if got := values[testTokenSpec().Env]; got != token {
		t.Fatalf("expected preserved token, got %q", got)
	}
}

func TestProvisionServiceTokensRotatesToken(t *testing.T) {
	oldCredentialID := "11111111-1111-1111-1111-111111111111"
	token := apiKeyForCredentialID(oldCredentialID)
	fake := newFakeAtomTokenServer(t, map[string]bool{token: true})
	defer fake.Close()

	output := filepath.Join(t.TempDir(), ".env.tokens")
	if err := os.WriteFile(output, []byte(testTokenSpec().Env+"="+token+"\n"), 0o600); err != nil {
		t.Fatalf("write existing token file: %v", err)
	}

	result, err := ProvisionServiceTokens(context.Background(), fake.Client(), TokenProvisionOptions{
		OutputPath: output,
		Rotate:     "journal",
		Specs:      []ServiceTokenSpec{testTokenSpec()},
	})
	if err != nil {
		t.Fatalf("provision tokens failed: %v", err)
	}
	if !containsString(result.Rotated, testTokenSpec().Env) {
		t.Fatalf("expected token to be rotated, got result %+v", result)
	}
	if !containsString(fake.revoked, oldCredentialID) {
		t.Fatalf("expected old credential to be revoked, got %v", fake.revoked)
	}
	values, err := readTokenEnvFile(output)
	if err != nil {
		t.Fatalf("read token env file: %v", err)
	}
	if got := values[testTokenSpec().Env]; got == "" || got == token {
		t.Fatalf("expected rotated token, got %q", got)
	}
}

func TestCredentialIDFromAPIKey(t *testing.T) {
	want := "11111111-2222-3333-4444-555555555555"
	got, ok := CredentialIDFromAPIKey(apiKeyForCredentialID(want))
	if !ok {
		t.Fatalf("expected credential id to parse")
	}
	if got != want {
		t.Fatalf("unexpected credential id: got %s want %s", got, want)
	}
	if _, ok := CredentialIDFromAPIKey("not-an-api-key"); ok {
		t.Fatalf("expected invalid token to be rejected")
	}
}

type fakeAtomTokenServer struct {
	t      *testing.T
	server *httptest.Server
	active map[string]bool

	created []map[string]any
	revoked []string
	nextID  int
}

func newFakeAtomTokenServer(t *testing.T, active map[string]bool) *fakeAtomTokenServer {
	t.Helper()
	fake := &fakeAtomTokenServer{
		t:      t,
		active: active,
	}
	if fake.active == nil {
		fake.active = map[string]bool{}
	}
	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))
	return fake
}

func (f *fakeAtomTokenServer) Close() {
	f.server.Close()
}

func (f *fakeAtomTokenServer) Client() *Client {
	return NewClient(Config{URL: f.server.URL, Token: "admin-token", Timeout: time.Second})
}

func (f *fakeAtomTokenServer) handle(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/auth/introspect":
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		_ = json.NewEncoder(w).Encode(IntrospectionResponse{Active: f.active[token], EntityID: "entity-1"})
	case atomGraphQLPath:
		f.handleGraphQL(w, r)
	default:
		f.t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}
}

func (f *fakeAtomTokenServer) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		f.t.Fatalf("unexpected GraphQL method: %s", r.Method)
	}
	var payload struct {
		Query     string         `json:"query"`
		Variables map[string]any `json:"variables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		f.t.Fatalf("decode GraphQL request: %v", err)
	}

	switch {
	case strings.Contains(payload.Query, "createApiKey"):
		input := payload.Variables["input"].(map[string]any)
		f.created = append(f.created, input)
		f.nextID++
		credentialID := credentialIDForIndex(f.nextID)
		key := apiKeyForCredentialID(credentialID)
		f.active[key] = true
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"createApiKey": APIKeyResponse{
					CredentialID: credentialID,
					Key:          key,
				},
			},
		})
	case strings.Contains(payload.Query, "revokeCredential"):
		credentialID := payload.Variables["credentialId"].(string)
		f.revoked = append(f.revoked, credentialID)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"revokeCredential": true},
		})
	default:
		f.t.Fatalf("unexpected GraphQL payload: %s", payload.Query)
	}
}

func testTokenSpec() ServiceTokenSpec {
	return ServiceTokenSpec{Name: "journal", Env: "MG_ATOM_TOKEN_JOURNAL", Description: "test journal token"}
}

func apiKeyForCredentialID(id string) string {
	return "atom_" + strings.ReplaceAll(id, "-", "") + "_" + strings.Repeat("a", 64)
}

func credentialIDForIndex(index int) string {
	return fmt.Sprintf("aaaaaaaa-aaaa-aaaa-aaaa-%012d", index)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertNoTempTokenFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read output directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".env.tokens-") {
			t.Fatalf("temporary token file was not removed: %s", entry.Name())
		}
	}
}
