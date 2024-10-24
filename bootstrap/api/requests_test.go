// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/stretchr/testify/assert"
)

var (
	channel1 = testsutil.GenerateUUID(&testing.T{})
	channel2 = testsutil.GenerateUUID(&testing.T{})
)

func TestAddReqValidation(t *testing.T) {
	cases := []struct {
		desc        string
		token       string
		externalID  string
		externalKey string
		channels    []string
		err         error
	}{
		{
			desc:        "valid request",
			token:       "token",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         nil,
		},
		{
			desc:        "empty token",
			token:       "",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "empty external ID",
			token:       "token",
			externalID:  "",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "empty external key",
			token:       "token",
			externalID:  "external-id",
			externalKey: "",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrBearerKey,
		},
		{
			desc:        "empty external key and external ID",
			token:       "token",
			externalID:  "",
			externalKey: "",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "empty channels",
			token:       "token",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{},
			err:         apiutil.ErrEmptyList,
		},
		{
			desc:        "empty channel value",
			token:       "token",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, ""},
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := addReq{
			token:          tc.token,
			ExternalID:     tc.externalID,
			ExternalSecret: tc.externalKey,
			Channels:       tc.channels,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestEntityReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "empty id",
			id:   "",
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := entityReq{
			id: tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "valid request",
			id:   "id",
			err:  nil,
		},
		{
			desc: "empty id",
			id:   "",
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			id: tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCertReqValidation(t *testing.T) {
	cases := []struct {
		desc    string
		clientID string
		err     error
	}{
		{
			desc:    "empty client id",
			clientID: "",
			err:     apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			clientID: tc.clientID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnReqValidation(t *testing.T) {
	cases := []struct {
		desc  string
		id    string
		token string

		err error
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
		req := updateConnReq{
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
		limit  uint64
		err    error
	}{
		{
			desc:   "too large limit",
			offset: 0,
			limit:  maxLimitSize + 1,
			err:    apiutil.ErrLimitSize,
		},
		{
			desc:   "default limit",
			offset: 0,
			limit:  defLimit,
			err:    nil,
		},
	}

	for _, tc := range cases {
		req := listReq{
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
