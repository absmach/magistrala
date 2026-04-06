// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package oauth2

import (
	"testing"

	"github.com/absmach/magistrala/users"
	"github.com/stretchr/testify/assert"
)

func TestNormalizeUser(t *testing.T) {
	cases := []struct {
		desc       string
		inputJSON  string
		provider   string
		wantUser   users.User
		wantErrStr string
	}{
		{
			desc: "valid user with standard keys",
			inputJSON: `{
				"id": "123",
				"given_name": "Jane",
				"family_name": "Doe",
				"email": "jane@example.com",
				"picture": "pic.jpg"
			}`,
			provider: "google",
			wantUser: users.User{
				ID:             "123",
				FirstName:      "Jane",
				LastName:       "Doe",
				Email:          "jane@example.com",
				ProfilePicture: "pic.jpg",
				Metadata:       users.Metadata{"oauth_provider": "google"},
			},
			wantErrStr: "",
		},
		{
			desc: "missing required fields",
			inputJSON: `{
				"given_name": "Jane"
			}`,
			provider:   "google",
			wantUser:   users.User{},
			wantErrStr: "missing required fields: id, last_name, email",
		},
		{
			desc:       "invalid JSON",
			inputJSON:  `{invalid json`,
			provider:   "google",
			wantUser:   users.User{},
			wantErrStr: "invalid character",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			user, err := NormalizeUser([]byte(tc.inputJSON), tc.provider)
			if tc.wantErrStr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrStr)
				assert.Equal(t, tc.wantUser, user)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.wantUser, user)
			}
		})
	}
}

func TestNormalizeProfile(t *testing.T) {
	cases := []struct {
		desc     string
		raw      map[string]any
		expected map[string]any
	}{
		{
			desc: "maps all variants to normalized keys",
			raw: map[string]any{
				"id":             "id123",
				"givenName":      "John",
				"familyName":     "Smith",
				"user_name":      "jsmith",
				"emailAddress":   "john@smith.com",
				"profilePicture": "pic.png",
			},
			expected: map[string]any{
				"id":         "id123",
				"first_name": "John",
				"last_name":  "Smith",
				"username":   "jsmith",
				"email":      "john@smith.com",
				"picture":    "pic.png",
			},
		},
		{
			desc:     "missing keys returns empty map",
			raw:      map[string]any{"foo": "bar"},
			expected: map[string]any{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := normalizeProfile(tc.raw)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestValidateUser(t *testing.T) {
	cases := []struct {
		desc    string
		user    normalizedUser
		wantErr string
	}{
		{
			desc: "valid user returns nil error",
			user: normalizedUser{
				ID:        "1",
				FirstName: "F",
				LastName:  "L",
				Email:     "e@example.com",
			},
			wantErr: "",
		},
		{
			desc: "missing id returns error",
			user: normalizedUser{
				FirstName: "F",
				LastName:  "L",
				Email:     "e@example.com",
			},
			wantErr: "missing required fields: id",
		},
		{
			desc:    "multiple missing fields returns all in error",
			user:    normalizedUser{},
			wantErr: "missing required fields: id, first_name, last_name, email",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := validateUser(tc.user)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, tc.wantErr, err.Error())
			}
		})
	}
}
