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
		domainID    string
		externalID  string
		externalKey string
		channels    []string
		err         error
	}{
		{
			desc:        "valid request",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         nil,
		},
		{
			desc:        "empty domain id",
			token:       "token",
			domainID:    "",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "empty token",
			token:       "",
			domainID:    "domain-id",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "empty external ID",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "",
			externalKey: "external-key",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "empty external key",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "external-id",
			externalKey: "",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrBearerKey,
		},
		{
			desc:        "empty external key and external ID",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "",
			externalKey: "",
			channels:    []string{channel1, channel2},
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "empty channels",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{},
			err:         apiutil.ErrEmptyList,
		},
		{
			desc:        "empty channel value",
			token:       "token",
			domainID:    "domain-id",
			externalID:  "external-id",
			externalKey: "external-key",
			channels:    []string{channel1, ""},
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := addReq{
			token:       tc.token,
			domainID:    tc.domainID,
			ExternalID:  tc.externalID,
			ExternalKey: tc.externalKey,
			Channels:    tc.channels,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestEntityReqValidation(t *testing.T) {
	cases := []struct {
		desc     string
		domainID string
		id       string
		err      error
	}{
		{
			desc:     "empty domain-id",
			domainID: "",
			id:       "id",
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "empty id",
			domainID: "domain-id",
			id:       "",
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := entityReq{
			domainID: tc.domainID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateReqValidation(t *testing.T) {
	cases := []struct {
		desc     string
		domainID string
		id       string
		err      error
	}{
		{
			desc:     "valid request",
			domainID: "domain-id",
			id:       "id",
			err:      nil,
		},
		{
			desc:     "empty domain-id",
			domainID: "",
			id:       "id",
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "empty id",
			domainID: "domain-id",
			id:       "",
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			id:       tc.id,
			domainID: tc.domainID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCertReqValidation(t *testing.T) {
	cases := []struct {
		desc     string
		domainID string
		thingID  string
		err      error
	}{
		{
			desc:     "empty domain id",
			domainID: "",
			thingID:  "thingID",
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "empty thing id",
			domainID: "domainID",
			thingID:  "",
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			thingID:  tc.thingID,
			domainID: tc.domainID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnReqValidation(t *testing.T) {
	cases := []struct {
		desc     string
		id       string
		token    string
		domainID string
		err      error
	}{
		{
			desc:     "empty token",
			token:    "",
			domainID: "domainID",
			id:       "id",
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "empty domain id",
			token:    "token",
			domainID: "",
			id:       "id",
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "empty id",
			token:    "token",
			domainID: "domainID",
			id:       "",
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateConnReq{
			token:    tc.token,
			id:       tc.id,
			domainID: tc.domainID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListReqValidation(t *testing.T) {
	cases := []struct {
		desc     string
		offset   uint64
		domainID string
		limit    uint64
		err      error
	}{
		{
			desc:     "empty domain id",
			domainID: "",
			offset:   0,
			limit:    1,
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "too large limit",
			domainID: "domainID",
			offset:   0,
			limit:    maxLimitSize + 1,
			err:      apiutil.ErrLimitSize,
		},
		{
			desc:     "default limit",
			domainID: "domainID",
			offset:   0,
			limit:    defLimit,
			err:      nil,
		},
	}

	for _, tc := range cases {
		req := listReq{
			token:    tc.token,
			offset:   tc.offset,
			limit:    tc.limit,
			domainID: tc.domainID,
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
		desc     string
		token    string
		domainID string
		id       string
		state    bootstrap.State
		err      error
	}{
		{
			desc:     "empty token",
			token:    "",
			domainID: "domainID",
			id:       "id",
			state:    bootstrap.State(1),
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "empty domain id",
			token:    "token",
			domainID: "",
			id:       "id",
			state:    bootstrap.State(1),
			err:      apiutil.ErrMissingDomainID,
		},
		{
			desc:     "empty id",
			token:    "token",
			domainID: "domainID",
			id:       "",
			state:    bootstrap.State(0),
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "invalid state",
			token:    "token",
			domainID: "domainID",
			id:       "id",
			state:    bootstrap.State(14),
			err:      apiutil.ErrBootstrapState,
		},
	}

	for _, tc := range cases {
		req := changeStateReq{
			token:    tc.token,
			id:       tc.id,
			State:    tc.state,
			domainID: tc.domainID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
