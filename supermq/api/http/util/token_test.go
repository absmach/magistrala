// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package util_test

import (
	"net/http"
	"testing"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/stretchr/testify/assert"
)

func TestExtractBearerToken(t *testing.T) {
	cases := []struct {
		desc    string
		request *http.Request
		token   string
	}{
		{
			desc: "valid bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"Bearer 123"},
				},
			},
			token: "123",
		},
		{
			desc: "invalid bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"123"},
				},
			},
			token: "",
		},
		{
			desc: "empty bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {""},
				},
			},
			token: "",
		},
		{
			desc: "empty header",
			request: &http.Request{
				Header: map[string][]string{},
			},
			token: "",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			token := apiutil.ExtractBearerToken(c.request)
			assert.Equal(t, c.token, token)
		})
	}
}

func TestExtractClientSecret(t *testing.T) {
	cases := []struct {
		desc    string
		request *http.Request
		token   string
	}{
		{
			desc: "valid bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"Client 123"},
				},
			},
			token: "123",
		},
		{
			desc: "invalid bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {"123"},
				},
			},
			token: "",
		},
		{
			desc: "empty bearer token",
			request: &http.Request{
				Header: map[string][]string{
					"Authorization": {""},
				},
			},
			token: "",
		},
		{
			desc: "empty header",
			request: &http.Request{
				Header: map[string][]string{},
			},
			token: "",
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			token := apiutil.ExtractClientSecret(c.request)
			assert.Equal(t, c.token, token)
		})
	}
}
