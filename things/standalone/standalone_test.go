// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/standalone"
	"github.com/stretchr/testify/assert"
)

const (
	email = "john.doe@example.com"
	token = "token"
)

func TestIdentify(t *testing.T) {
	svc := standalone.NewAuthService(email, token)

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "identify non-existing user",
			token: "non-existing",
			id:    "",
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "identify existing user",
			token: token,
			id:    email,
			err:   nil,
		},
	}

	for _, tc := range cases {
		id, err := svc.Identify(context.Background(), &mainflux.Token{Value: tc.token})
		assert.Equal(t, tc.id, id.GetEmail(), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.id, id.GetEmail()))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
	}
}

func TestIssue(t *testing.T) {
	svc := standalone.NewAuthService(email, token)

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "issue key",
			token: token,
			id:    token,
			err:   nil,
		},
		{
			desc:  "issue key with an invalid token",
			token: "non-existing",
			id:    "",
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		id, err := svc.Issue(context.Background(), &mainflux.IssueReq{Id: tc.id, Email: tc.token, Type: 0})
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s, got %s", tc.desc, tc.err, err))
	}
}
