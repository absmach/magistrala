// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/stretchr/testify/assert"
)

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
		res := tc.key.Expired()
		assert.Equal(t, tc.expired, res, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.expired, res))
	}
}
