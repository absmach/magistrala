// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"testing"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/eventlogs"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/stretchr/testify/assert"
)

var (
	token        = "token"
	limit uint64 = 10
)

func TestListEventsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  listEventsReq
		err  error
	}{
		{
			desc: "valid",
			req: listEventsReq{
				token: token,
				page: eventlogs.Page{
					ID:         testsutil.GenerateUUID(t),
					EntityType: auth.UserType,
					Limit:      limit,
				},
			},
			err: nil,
		},
		{
			desc: "missing token",
			req: listEventsReq{
				page: eventlogs.Page{
					ID:         testsutil.GenerateUUID(t),
					EntityType: auth.UserType,
					Limit:      limit,
				},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "missing id",
			req: listEventsReq{
				token: token,
				page: eventlogs.Page{
					EntityType: auth.UserType,
					Limit:      limit,
				},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "missing entity type",
			req: listEventsReq{
				token: token,
				page: eventlogs.Page{
					ID:    testsutil.GenerateUUID(t),
					Limit: limit,
				},
			},
			err: apiutil.ErrMissingEntityType,
		},
		{
			desc: "invalid entity type",
			req: listEventsReq{
				token: token,
				page: eventlogs.Page{
					ID:         testsutil.GenerateUUID(t),
					EntityType: "invalid",
					Limit:      limit,
				},
			},
			err: apiutil.ErrInvalidEntityType,
		},
		{
			desc: "invalid limit size",
			req: listEventsReq{
				token: token,
				page: eventlogs.Page{
					ID:         testsutil.GenerateUUID(t),
					EntityType: auth.UserType,
					Limit:      maxLimitSize + 1,
				},
			},
			err: apiutil.ErrLimitSize,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := c.req.validate()
			assert.Equal(t, c.err, err)
		})
	}
}
