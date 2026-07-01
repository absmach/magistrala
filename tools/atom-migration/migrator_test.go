// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"testing"

	"github.com/google/uuid"
)

func TestGroupRoleBlockPlansUseAtomSupportedScopes(t *testing.T) {
	scope := roleScope{prefix: "groups", scopeMode: "group"}
	roleID := "role-1"
	groupID := "11111111-1111-1111-1111-111111111111"
	tenantID := "22222222-2222-2222-2222-222222222222"

	cases := []struct {
		name       string
		action     string
		scopeMode  string
		objectKind any
		objectType any
		objectID   any
		groupID    any
	}{
		{
			name: "direct device action", action: "client_read",
			scopeMode: "group_direct_objects", objectKind: "entity", objectType: "entity:device", groupID: groupID,
		},
		{
			name: "descendant device action", action: "subgroup_client_set_parent_group",
			scopeMode: "group_descendant_objects", objectKind: "entity", objectType: "entity:device", groupID: groupID,
		},
		{
			name: "direct channel action", action: "channel_publish",
			scopeMode: "group_direct_objects", objectKind: "resource", objectType: "resource:channel", groupID: groupID,
		},
		{
			name: "descendant channel action", action: "subgroup_channel_subscribe",
			scopeMode: "group_descendant_objects", objectKind: "resource", objectType: "resource:channel", groupID: groupID,
		},
		{
			name: "descendant group action", action: "subgroup_set_child",
			scopeMode: "group_descendant_groups", groupID: groupID,
		},
		{
			name: "self group action", action: "manage_role",
			scopeMode: "object", objectKind: "group", objectID: groupID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plans := scope.blockPlans(roleID, groupID, tenantID, tc.action)
			if len(plans) != 1 {
				t.Fatalf("expected one plan, got %d", len(plans))
			}
			got := plans[0]
			if got.ScopeMode != tc.scopeMode {
				t.Fatalf("scope mode = %q, want %q", got.ScopeMode, tc.scopeMode)
			}
			if got.TenantID != tenantID {
				t.Fatalf("tenant = %q, want %q", got.TenantID, tenantID)
			}
			if got.ObjectKind != tc.objectKind || got.ObjectType != tc.objectType || got.ObjectID != tc.objectID || got.GroupID != tc.groupID {
				t.Fatalf("plan = %+v, want objectKind=%v objectType=%v objectID=%v groupID=%v", got, tc.objectKind, tc.objectType, tc.objectID, tc.groupID)
			}
		})
	}
}

func TestMapActionPreservesChannelPublishSubscribeVariants(t *testing.T) {
	cases := map[string]string{
		"channel_publish":            actionPublish,
		"subgroup_channel_publish":   actionPublish,
		"channel_subscribe":          actionSubscribe,
		"subgroup_channel_subscribe": actionSubscribe,
	}
	for raw, want := range cases {
		got, ok := mapAction(raw)
		if !ok {
			t.Fatalf("mapAction(%q) returned not ok", raw)
		}
		if got != want {
			t.Fatalf("mapAction(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestClientSharedKeyCredentialIDIsStable(t *testing.T) {
	clientID := "11111111-1111-1111-1111-111111111111"
	got := clientSharedKeyCredentialID(clientID)
	if got == "" {
		t.Fatal("expected credential id")
	}
	if again := clientSharedKeyCredentialID(clientID); again != got {
		t.Fatalf("credential id is not stable: got %q then %q", got, again)
	}
	if oldAPIKeyID := derivedUUID("devcred", clientID); oldAPIKeyID == got {
		t.Fatalf("shared-key credential id must not collide with legacy API-key id %q", oldAPIKeyID)
	}
}

func TestNewSharedKeyMaterialEncryptsRecoverableSecret(t *testing.T) {
	secret := "client-secret"
	credentialID := clientSharedKeyCredentialID("11111111-1111-1111-1111-111111111111")
	cfg := config{
		AtomKeyEncryptionKey:   []byte("0123456789abcdef0123456789abcdef"),
		AtomKeyEncryptionKeyID: "local:test",
	}

	material, err := newSharedKeyMaterial(credentialID, secret, cfg)
	if err != nil {
		t.Fatalf("new shared-key material: %v", err)
	}
	if material.Hash == "" || len(material.Ciphertext) == 0 || len(material.Nonce) != aeadNonceLen || len(material.LookupHash) != 32 {
		t.Fatalf("unexpected material: %+v", material)
	}
	if material.KeyID != cfg.AtomKeyEncryptionKeyID || material.EncAlg != sharedKeyAEADAlg {
		t.Fatalf("unexpected encryption metadata: %+v", material)
	}

	block, err := aes.NewCipher(cfg.AtomKeyEncryptionKey)
	if err != nil {
		t.Fatalf("cipher: %v", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	credUUID, err := uuid.Parse(credentialID)
	if err != nil {
		t.Fatalf("credential id: %v", err)
	}
	plaintext, err := aead.Open(nil, material.Nonce, material.Ciphertext, credUUID[:])
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(plaintext) != secret {
		t.Fatalf("plaintext = %q, want %q", plaintext, secret)
	}
}
