// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package jwt_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const secret = "test"

func key() auth.Key {
	exp := time.Now().UTC().Add(10 * time.Minute).Round(time.Second)
	return auth.Key{
		ID:        "66af4a67-3823-438a-abd7-efdb613eaef6",
		Type:      auth.AccessKey,
		Issuer:    "magistrala.auth",
		Subject:   "66af4a67-3823-438a-abd7-efdb613eaef6",
		IssuedAt:  time.Now().UTC().Add(-10 * time.Second).Round(time.Second),
		ExpiresAt: exp,
	}
}

func TestIssue(t *testing.T) {
	tokenizer := jwt.New([]byte(secret))

	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "issue new token",
			key:  key(),
			err:  nil,
		},
	}

	for _, tc := range cases {
		tkn, err := tokenizer.Issue(tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err != nil {
			assert.NotEmpty(t, tkn, fmt.Sprintf("%s expected token, got empty string", tc.desc))
		}
	}
}

func TestParse(t *testing.T) {
	tokenizer := jwt.New([]byte(secret))

	token, err := tokenizer.Issue(key())
	require.Nil(t, err, fmt.Sprintf("issuing key expected to succeed: %s", err))

	apiKey := key()
	apiKey.Type = auth.APIKey
	apiKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	apiToken, err := tokenizer.Issue(apiKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	expKey := key()
	expKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expToken, err := tokenizer.Issue(expKey)
	require.Nil(t, err, fmt.Sprintf("issuing expired key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   auth.Key
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
			key:   auth.Key{},
			token: "invalid",
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "parse expired key",
			key:   auth.Key{},
			token: expToken,
			err:   jwt.ErrExpiry,
		},
		{
			desc:  "parse expired API key",
			key:   apiKey,
			token: apiToken,
			err:   jwt.ErrExpiry,
		},
	}

	for _, tc := range cases {
		key, err := tokenizer.Parse(tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.key, key, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.key, key))
		}
	}
}
