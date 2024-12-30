// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"testing"

	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
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
				domainID:    testsutil.GenerateUUID(t),
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
				domainID:    testsutil.GenerateUUID(t),
				Name:        "name",
				ExternalID:  "",
				ExternalKey: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty domain id",
			req: provisionReq{
				token:       "token",
				domainID:    "",
				Name:        "name",
				ExternalID:  testsutil.GenerateUUID(t),
				ExternalKey: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrMissingDomainID,
		},
		{
			desc: "empty external key",
			req: provisionReq{
				token:       "token",
				domainID:    testsutil.GenerateUUID(t),
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
				token:    "token",
				domainID: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "empty token",
			req: mappingReq{
				token:    "",
				domainID: testsutil.GenerateUUID(t),
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "empty domain id",
			req: mappingReq{
				token:    "token",
				domainID: "",
			},
			err: apiutil.ErrMissingDomainID,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected `%v` got `%v`", tc.desc, tc.err, err))
	}
}
