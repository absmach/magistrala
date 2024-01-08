// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestProvisioReq(t *testing.T) {
	cases := []struct {
		desc string
		req  provisionReq
		err  error
	}{
		{
			desc: "valid request",
			req: provisionReq{
				token:       "token",
				Name:        "name",
				ExternalID:  testsutil.GenerateUUID(t),
				ExternalKey: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "empty external id",
			req: provisionReq{
				token:       "token",
				Name:        "name",
				ExternalID:  "",
				ExternalKey: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty external key",
			req: provisionReq{
				token:       "token",
				Name:        "name",
				ExternalID:  testsutil.GenerateUUID(t),
				ExternalKey: "",
			},
			err: apiutil.ErrBearerKey,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", tc.desc, tc.err, err))
	}
}

func TestMappingReq(t *testing.T) {
	cases := []struct {
		desc string
		req  mappingReq
		err  error
	}{
		{
			desc: "valid request",
			req: mappingReq{
				token: "token",
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: mappingReq{
				token: "",
			},
			err: apiutil.ErrBearerToken,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", tc.desc, tc.err, err))
	}
}
