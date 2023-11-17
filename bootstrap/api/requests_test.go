// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/stretchr/testify/assert"
)

func TestAddReqValidation(t *testing.T) {
	cases := []struct {
		desc        string
		token       string
		externalID  string
		externalKey string
		err         error
	}{
		{
			desc:        "empty key",
			token:       "",
			externalID:  "external-id",
			externalKey: "external-key",
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "empty external ID",
			token:       "token",
			externalID:  "",
			externalKey: "external-key",
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "empty external key",
			token:       "token",
			externalID:  "external-id",
			externalKey: "",
			err:         apiutil.ErrBearerKey,
		},
	}

	for _, tc := range cases {
		req := addReq{
			token:       tc.token,
			ExternalID:  tc.externalID,
			ExternalKey: tc.externalKey,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestEntityReqValidation(t *testing.T) {
	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "empty token",
			token: "",
			id:    "id",
			err:   apiutil.ErrBearerToken,
		},
		{
			desc:  "empty id",
			token: "token",
			id:    "",
			err:   apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := entityReq{
			token: tc.token,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateReqValidation(t *testing.T) {
	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "empty token",
			token: "",
			id:    "id",
			err:   apiutil.ErrBearerToken,
		},
		{
			desc:  "empty id",
			token: "token",
			id:    "",
			err:   apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			token: tc.token,
			id:    tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCertReqValidation(t *testing.T) {
	cases := []struct {
		desc    string
		token   string
		thingID string
		err     error
	}{
		{
			desc:    "empty token",
			token:   "",
			thingID: "thingID",
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:    "empty thing id",
			token:   "token",
			thingID: "",
			err:     apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			token:   tc.token,
			thingID: tc.thingID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnReqValidation(t *testing.T) {
	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "empty token",
			token: "",
			id:    "id",
			err:   apiutil.ErrBearerToken,
		},
		{
			desc:  "empty id",
			token: "token",
			id:    "",
			err:   apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			token: tc.token,
			id:    tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListReqValidation(t *testing.T) {
	cases := []struct {
		desc   string
		offset uint64
		token  string
		limit  uint64
		err    error
	}{
		{
			desc:   "empty token",
			token:  "",
			offset: 0,
			limit:  1,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "too large limit",
			token:  "token",
			offset: 0,
			limit:  maxLimitSize + 1,
			err:    apiutil.ErrLimitSize,
		},
		{
			desc:   "default limit",
			token:  "token",
			offset: 0,
			limit:  defLimit,
			err:    nil,
		},
	}

	for _, tc := range cases {
		req := listReq{
			token:  tc.token,
			offset: tc.offset,
			limit:  tc.limit,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestBootstrapReqValidation(t *testing.T) {
	cases := []struct {
		desc      string
		externKey string
		externID  string
		err       error
	}{
		{
			desc:      "empty external key",
			externKey: "",
			externID:  "id",
			err:       apiutil.ErrBearerKey,
		},
		{
			desc:      "empty external id",
			externKey: "key",
			externID:  "",
			err:       apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := bootstrapReq{
			id:  tc.externID,
			key: tc.externKey,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeStateReqValidation(t *testing.T) {
	cases := []struct {
		desc  string
		token string
		id    string
		state bootstrap.State
		err   error
	}{
		{
			desc:  "empty token",
			token: "",
			id:    "id",
			state: bootstrap.State(1),
			err:   apiutil.ErrBearerToken,
		},
		{
			desc:  "empty id",
			token: "token",
			id:    "",
			state: bootstrap.State(0),
			err:   apiutil.ErrMissingID,
		},
		{
			desc:  "invalid state",
			token: "token",
			id:    "id",
			state: bootstrap.State(14),
			err:   apiutil.ErrBootstrapState,
		},
	}

	for _, tc := range cases {
		req := changeStateReq{
			token: tc.token,
			id:    tc.id,
			State: tc.state,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
