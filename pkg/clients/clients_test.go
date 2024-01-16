// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package clients

import (
	"testing"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidateClient(t *testing.T) {
	cases := []struct {
		desc     string
		identity string
		err      error
	}{
		{
			desc:     "valid identity",
			identity: "user@example.com",
			err:      nil,
		},
		{
			desc:     "invalid identity",
			identity: "user@example",
			err:      errors.ErrMalformedEntity,
		},
		{
			desc: "empty identity",
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			client := Client{
				Credentials: Credentials{
					Identity: c.identity,
				},
			}
			err := client.Validate()
			assert.Equal(t, c.err, err, "ValidateClient() error = %v, expected %v", err, c.err)
		})
	}
}

func TestIsEmail(t *testing.T) {
	cases := []struct {
		email    string
		expected bool
	}{
		{
			email:    "test@example.com",
			expected: true,
		},
		{
			email:    "test-test@example.com",
			expected: true,
		},
		{
			email:    "test.test@example.com",
			expected: true,
		},
		{
			email:    "test_test@example.com",
			expected: true,
		},
		{
			email:    "test@",
			expected: false,
		},
		{
			email:    "@",
			expected: false,
		},
		{
			email:    "test.example.com",
			expected: false,
		},
		{
			email:    "@example.com",
			expected: false,
		},
		{
			email:    "test@example",
			expected: false,
		},
		{
			email:    "test@example.",
			expected: false,
		},
		{
			email:    "test@.com",
			expected: false,
		},
		{
			email:    "test@.example.com",
			expected: false,
		},
		{
			email:    "test@example.com.",
			expected: false,
		},
		{
			email:    "test@example.",
			expected: false,
		},
		{
			email:    "test@subdomain.example.com",
			expected: true,
		},
		{
			email:    "test@subdomain-example.com",
			expected: true,
		},
		{
			email:    "test@subdomain_example.com",
			expected: true,
		},
		{
			email:    "@subdomain.example.com",
			expected: false,
		},
		{
			email:    "test@subdomain.subdomain.example.com",
			expected: true,
		},
		{
			email:    "test@subdomain..com",
			expected: false,
		},
		{
			email:    "test@subdomain..example.com",
			expected: false,
		},
		{
			email:    "test@subdomain.example..com",
			expected: false,
		},
		{
			email:    "test@subdomain.example.com.",
			expected: false,
		},
		{
			email:    "test@subdomain.example.com..",
			expected: false,
		},
	}

	for _, c := range cases {
		isValid := isEmail(c.email)
		if isValid != c.expected {
			t.Errorf("Expected isEmail(%s) to be %v, but got %v", c.email, c.expected, isValid)
		}
	}
}
