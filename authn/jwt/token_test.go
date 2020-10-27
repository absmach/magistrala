// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/authn"
	"github.com/mainflux/mainflux/authn/jwt"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test"

func key() authn.Key {
	exp := time.Now().UTC().Add(10 * time.Minute).Round(time.Second)
	return authn.Key{
		ID:        "id",
		Type:      authn.UserKey,
		Subject:   "user@email.com",
		IssuerID:  "",
		IssuedAt:  time.Now().UTC().Add(-10 * time.Second).Round(time.Second),
		ExpiresAt: exp,
	}
}

func TestIssue(t *testing.T) {
	tokenizer := jwt.New(secret)

	cases := []struct {
		desc string
		key  authn.Key
		err  error
	}{
		{
			desc: "issue new token",
			key:  key(),
			err:  nil,
		},
	}

	for _, tc := range cases {
		_, err := tokenizer.Issue(tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
	}
}

func TestParse(t *testing.T) {
	tokenizer := jwt.New(secret)

	token, err := tokenizer.Issue(key())
	require.Nil(t, err, fmt.Sprintf("issuing key expected to succeed: %s", err))

	userKey := key()
	userKey.Type = authn.APIKey
	userKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	userToken, err := tokenizer.Issue(userKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	expKey := key()
	expKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expToken, err := tokenizer.Issue(expKey)
	require.Nil(t, err, fmt.Sprintf("issuing expired key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   authn.Key
		token string
		err   error
	}{
		{
			desc:  "parse valid key",
			key:   key(),
			token: token,
			err:   nil,
		},
		{
			desc:  "parse ivalid key",
			key:   authn.Key{},
			token: "invalid",
			err:   authn.ErrUnauthorizedAccess,
		},

		{
			desc:  "parse expired key",
			key:   authn.Key{},
			token: expToken,
			err:   authn.ErrKeyExpired,
		},
		{
			desc:  "parse expired user key",
			key:   userKey,
			token: userToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		key, err := tokenizer.Parse(tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.key, key, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.key, key))
	}
}
