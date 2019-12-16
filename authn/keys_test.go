// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package authn_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/authn"
	"github.com/stretchr/testify/assert"
)

func TestExpired(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute)
	exp1 := time.Now()
	cases := []struct {
		desc    string
		key     authn.Key
		expired bool
	}{
		{
			desc: "not expired key",
			key: authn.Key{
				IssuedAt:  time.Now(),
				ExpiresAt: exp,
			},
			expired: false,
		},
		{
			desc: "expired key",
			key: authn.Key{
				IssuedAt:  time.Now().UTC().Add(2 * time.Minute),
				ExpiresAt: exp1,
			},
			expired: true,
		},
		{
			desc: "key with no expiration date",
			key: authn.Key{
				IssuedAt: time.Now(),
			},
			expired: true,
		},
	}

	for _, tc := range cases {
		res := tc.key.Expired()
		assert.Equal(t, tc.expired, res, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.expired, res))
	}
}
