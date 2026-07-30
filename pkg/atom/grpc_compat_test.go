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
	commonv1 "github.com/absmach/magistrala/api/grpc/common/v1"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
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

type fakeClientsCompatClient struct {
	fakePolicyClient
}

func (f *fakeClientsCompatClient) LoginSharedKey(context.Context, string, string) (LoginResponse, error) {
	return LoginResponse{}, nil
}

func TestAtomClientsCompatAuthenticatesBasicSharedKeyWithAtomLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/login" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var got LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode login request: %v", err)
		}
		if got.Identifier != testEntityID || got.Secret != testDeviceSecret || got.Kind != "shared_key" {
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
		t.Fatalf("authenticate basic shared key: %v", err)
	}
	if !res.GetAuthenticated() || res.GetId() != testEntityID {
		t.Fatalf("unexpected response: %+v", res)
	}
	if fallback.called {
		t.Fatal("token fallback should not be called after successful Atom shared-key login")
	}
}

func TestAtomClientsCompatFallsBackToBearerTokenWhenBasicSharedKeyRejected(t *testing.T) {
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

func TestAtomClientsCompatDoesNotHideAtomSharedKeyLoginFailures(t *testing.T) {
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

func TestAtomClientsCompatRemoveConnectionsDeletesConnectionPolicies(t *testing.T) {
	client := &fakeClientsCompatClient{
		fakePolicyClient: fakePolicyClient{
			capIDs: map[string]string{
				atomActionPublish:   "cap-publish",
				atomActionSubscribe: "cap-subscribe",
			},
			policies: []DirectPolicy{
				{
					ID:          "delete-publish",
					TenantID:    testDomainID,
					SubjectKind: atomObjectKindEntity,
					SubjectID:   testDeviceID,
					PermissionBlock: PermissionBlock{
						ID:         "publish-block",
						ScopeMode:  atomScopeModeObject,
						ObjectKind: atomObjectKindResource,
						ObjectType: "resource:channel",
						ObjectID:   "channel-1",
						Actions:    []Capability{{ID: "cap-publish"}},
					},
				},
				{
					ID:          "delete-subscribe",
					TenantID:    testDomainID,
					SubjectKind: atomObjectKindEntity,
					SubjectID:   testDeviceID,
					PermissionBlock: PermissionBlock{
						ID:         "subscribe-block",
						ScopeMode:  atomScopeModeObject,
						ObjectKind: atomObjectKindResource,
						ObjectType: "resource:channel",
						ObjectID:   "channel-1",
						Actions:    []Capability{{ID: "cap-subscribe"}},
					},
				},
				{
					ID:          "keep-other-channel",
					TenantID:    testDomainID,
					SubjectKind: atomObjectKindEntity,
					SubjectID:   testDeviceID,
					PermissionBlock: PermissionBlock{
						ID:         "other-block",
						ScopeMode:  atomScopeModeObject,
						ObjectKind: atomObjectKindResource,
						ObjectType: "resource:channel",
						ObjectID:   "other-channel",
						Actions:    []Capability{{ID: "cap-publish"}},
					},
				},
			},
		},
	}
	compat := AtomClientsCompat{Client: client}

	res, err := compat.RemoveConnections(context.Background(), &commonv1.RemoveConnectionsReq{
		Connections: []*commonv1.Connection{
			{
				ClientId:  testDeviceID,
				ChannelId: "channel-1",
				DomainId:  testDomainID,
				Type:      uint32(connections.Publish),
			},
			{
				ClientId:  testDeviceID,
				ChannelId: "channel-1",
				DomainId:  testDomainID,
				Type:      uint32(connections.Subscribe),
			},
		},
	})
	if err != nil {
		t.Fatalf("remove connections: %v", err)
	}
	if !res.GetOk() {
		t.Fatal("expected ok response")
	}
	if len(client.directPolicyQueries) != 2 {
		t.Fatalf("unexpected direct policy query count: %d", len(client.directPolicyQueries))
	}
	for _, q := range client.directPolicyQueries {
		if q.TenantID != testDomainID || q.SubjectKind != atomObjectKindEntity || q.SubjectID != testDeviceID {
			t.Fatalf("unexpected direct policy query: %+v", q)
		}
	}
	if len(client.deleted) != 2 || client.deleted[0] != "delete-publish" || client.deleted[1] != "delete-subscribe" {
		t.Fatalf("unexpected deleted policies: %+v", client.deleted)
	}
}
