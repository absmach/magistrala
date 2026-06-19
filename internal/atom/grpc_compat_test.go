// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	clientsv1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
)

type recordingAuthn struct {
	called  bool
	token   string
	session smqauthn.Session
	err     error
}

func (r *recordingAuthn) Authenticate(_ context.Context, token string) (smqauthn.Session, error) {
	r.called = true
	r.token = token
	return r.session, r.err
}

func TestAtomClientsCompatAuthenticatesBasicPasswordWithAtomLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/login" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var got LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode login request: %v", err)
		}
		if got.Identifier != testEntityID || got.Secret != testDeviceSecret || got.Kind != "password" {
			t.Fatalf("unexpected login request: %+v", got)
		}
		_ = json.NewEncoder(w).Encode(LoginResponse{
			Token:     "jwt",
			EntityID:  testEntityID,
			SessionID: "session-1",
			ExpiresAt: time.Now().Add(time.Hour),
		})
	}))
	defer srv.Close()

	fallback := &recordingAuthn{}
	compat := NewClientsCompat(fallback, NewClient(Config{URL: srv.URL, Timeout: time.Second}))
	token := smqauthn.AuthPack(smqauthn.BasicAuth, testEntityID, testDeviceSecret)

	res, err := compat.Authenticate(context.Background(), &clientsv1.AuthnReq{Token: token})
	if err != nil {
		t.Fatalf("authenticate basic password: %v", err)
	}
	if !res.GetAuthenticated() || res.GetId() != testEntityID {
		t.Fatalf("unexpected response: %+v", res)
	}
	if fallback.called {
		t.Fatal("token fallback should not be called after successful Atom password login")
	}
}

func TestAtomClientsCompatFallsBackToBearerTokenWhenBasicPasswordRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
	}))
	defer srv.Close()

	fallback := &recordingAuthn{session: smqauthn.Session{UserID: "entity-2"}}
	compat := NewClientsCompat(fallback, NewClient(Config{URL: srv.URL, Timeout: time.Second}))
	token := smqauthn.AuthPack(smqauthn.BasicAuth, testEntityID, "atom_token")

	res, err := compat.Authenticate(context.Background(), &clientsv1.AuthnReq{Token: token})
	if err != nil {
		t.Fatalf("authenticate fallback token: %v", err)
	}
	if !fallback.called || fallback.token != "atom_token" {
		t.Fatalf("unexpected fallback call: called=%v token=%q", fallback.called, fallback.token)
	}
	if !res.GetAuthenticated() || res.GetId() != "entity-2" {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestAtomClientsCompatDoesNotHideAtomPasswordLoginFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "atom unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	fallback := &recordingAuthn{session: smqauthn.Session{UserID: "entity-2"}}
	compat := NewClientsCompat(fallback, NewClient(Config{URL: srv.URL, Timeout: time.Second}))
	token := smqauthn.AuthPack(smqauthn.BasicAuth, testEntityID, testDeviceSecret)

	_, err := compat.Authenticate(context.Background(), &clientsv1.AuthnReq{Token: token})
	if err == nil {
		t.Fatal("expected Atom login failure")
	}
	if fallback.called {
		t.Fatal("token fallback should not be called for non-authentication Atom failures")
	}
}
