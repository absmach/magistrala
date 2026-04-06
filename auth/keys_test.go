// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/stretchr/testify/assert"
)

func TestKeyTypeString(t *testing.T) {
	cases := []struct {
		desc     string
		keyType  auth.KeyType
		expected string
	}{
		{
			desc:     "Access key type",
			keyType:  auth.AccessKey,
			expected: "access",
		},
		{
			desc:     "Refresh key type",
			keyType:  auth.RefreshKey,
			expected: "refresh",
		},
		{
			desc:     "Recovery key type",
			keyType:  auth.RecoveryKey,
			expected: "recovery",
		},
		{
			desc:     "API key type",
			keyType:  auth.APIKey,
			expected: "API",
		},
		{
			desc:     "Personal access token type",
			keyType:  auth.PersonalAccessToken,
			expected: "pat",
		},
		{
			desc:     "Invitation key type",
			keyType:  auth.InvitationKey,
			expected: "unknown",
		},
		{
			desc:     "Unknown key type",
			keyType:  auth.KeyType(100),
			expected: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.keyType.String()
			assert.Equal(t, tc.expected, got, "String() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestKeyTypeValidate(t *testing.T) {
	cases := []struct {
		desc     string
		keyType  auth.KeyType
		expected bool
	}{
		{
			desc:     "Valid access key",
			keyType:  auth.AccessKey,
			expected: true,
		},
		{
			desc:     "Valid refresh key",
			keyType:  auth.RefreshKey,
			expected: true,
		},
		{
			desc:     "Valid recovery key",
			keyType:  auth.RecoveryKey,
			expected: true,
		},
		{
			desc:     "Valid API key",
			keyType:  auth.APIKey,
			expected: true,
		},
		{
			desc:     "Valid personal access token",
			keyType:  auth.PersonalAccessToken,
			expected: true,
		},
		{
			desc:     "Valid invitation key",
			keyType:  auth.InvitationKey,
			expected: true,
		},
		{
			desc:     "Invalid key type (too large)",
			keyType:  auth.KeyType(100),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.keyType.Validate()
			assert.Equal(t, tc.expected, got, "Validate() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestRoleString(t *testing.T) {
	cases := []struct {
		desc     string
		role     auth.Role
		expected string
	}{
		{
			desc:     "User role",
			role:     auth.UserRole,
			expected: "user",
		},
		{
			desc:     "Admin role",
			role:     auth.AdminRole,
			expected: "admin",
		},
		{
			desc:     "Unknown role",
			role:     auth.Role(100),
			expected: "unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.role.String()
			assert.Equal(t, tc.expected, got, "String() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestRoleValidate(t *testing.T) {
	cases := []struct {
		desc     string
		role     auth.Role
		expected bool
	}{
		{
			desc:     "Valid user role",
			role:     auth.UserRole,
			expected: true,
		},
		{
			desc:     "Valid admin role",
			role:     auth.AdminRole,
			expected: true,
		},
		{
			desc:     "Invalid role (zero)",
			role:     auth.Role(0),
			expected: false,
		},
		{
			desc:     "Invalid role (too large)",
			role:     auth.Role(100),
			expected: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := tc.role.Validate()
			assert.Equal(t, tc.expected, got, "Validate() = %v, expected %v", got, tc.expected)
		})
	}
}

func TestKeyString(t *testing.T) {
	key := auth.Key{
		ID:        "test-id",
		Type:      auth.APIKey,
		Issuer:    "test-issuer",
		Subject:   "test-subject",
		Role:      auth.UserRole,
		IssuedAt:  time.Now().UTC().Round(time.Second),
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour).Round(time.Second),
	}

	str := key.String()
	assert.NotEmpty(t, str, "String() should return non-empty string")
	assert.Contains(t, str, "test-id", "String() should contain ID")
	assert.Contains(t, str, "test-issuer", "String() should contain Issuer")
	assert.Contains(t, str, "test-subject", "String() should contain Subject")
	assert.Contains(t, str, "API", "String() should contain Type")
	assert.Contains(t, str, "user", "String() should contain Role")
}

func TestExpired(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	exp1 := time.Now()
	cases := []struct {
		desc    string
		key     auth.Key
		expired bool
	}{
		{
			desc: "not expired key",
			key: auth.Key{
				IssuedAt:  time.Now(),
				ExpiresAt: exp,
			},
			expired: false,
		},
		{
			desc: "expired key",
			key: auth.Key{
				IssuedAt:  time.Now().UTC().Add(2 * time.Minute),
				ExpiresAt: exp1,
			},
			expired: true,
		},
		{
			desc: "user key with no expiration date",
			key: auth.Key{
				IssuedAt: time.Now(),
			},
			expired: true,
		},
		{
			desc: "API key with no expiration date",
			key: auth.Key{
				IssuedAt: time.Now(),
				Type:     auth.APIKey,
			},
			expired: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			res := tc.key.Expired()
			assert.Equal(t, tc.expired, res, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.expired, res))
		})
	}
}
