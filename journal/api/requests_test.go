// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"testing"

	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/journal"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

var (
	token        = "token"
	limit uint64 = 10
)

func TestRetrieveJournalsReqValidate(t *testing.T) {
	cases := []struct {
		desc string
		req  retrieveJournalsReq
		err  error
	}{
		{
			desc: "valid",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:      limit,
					EntityID:   "id",
					EntityType: journal.UserEntity,
				},
			},
			err: nil,
		},
		{
			desc: "missing token",
			req: retrieveJournalsReq{
				page: journal.Page{
					Limit:      limit,
					EntityID:   "id",
					EntityType: journal.UserEntity,
				},
			},
			err: apiutil.ErrBearerToken,
		},
		{
			desc: "invalid limit size",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:      api.DefLimit + 1,
					EntityID:   "id",
					EntityType: journal.UserEntity,
				},
			},
			err: apiutil.ErrLimitSize,
		},
		{
			desc: "invalid sorting direction",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:      limit,
					Direction:  "invalid",
					EntityID:   "id",
					EntityType: journal.UserEntity,
				},
			},
			err: apiutil.ErrInvalidDirection,
		},
		{
			desc: "valid id and entity type",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:      limit,
					EntityID:   "id",
					EntityType: journal.UserEntity,
				},
			},
			err: nil,
		},
		{
			desc: "valid id and empty entity type",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:    limit,
					EntityID: "id",
				},
			},
			err: nil,
		},
		{
			desc: "empty id and empty entity type",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit: limit,
				},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty id and valid entity type",
			req: retrieveJournalsReq{
				token: token,
				page: journal.Page{
					Limit:      limit,
					EntityType: journal.UserEntity,
				},
			},
			err: apiutil.ErrMissingID,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			err := c.req.validate()
			assert.Equal(t, c.err, err)
		})
	}
}
