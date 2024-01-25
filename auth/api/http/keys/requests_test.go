// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/stretchr/testify/assert"
)

var valid = "valid"

func TestIssueKeyReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  issueKeyReq
		err  error
	}{
		{
			desc: "valid request",
			req: issueKeyReq{
				token: valid,
				Type:  auth.AccessKey,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: issueKeyReq{
				token: "",
				Type:  auth.AccessKey,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "invalid key type",
			req: issueKeyReq{
				token: valid,
				Type:  auth.KeyType(100),
			},
			err: apiutil.ErrInvalidAPIKey,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err)
	}
}

func TestKeyReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  keyReq
		err  error
	}{
		{
			desc: "valid request",
			req: keyReq{
				token: valid,
				id:    valid,
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: keyReq{
				token: "",
				id:    valid,
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty id",
			req: keyReq{
				token: valid,
				id:    "",
			},
			err: apiutil.ErrMissingID,
		},
	}
	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err)
	}
}
