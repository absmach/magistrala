// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"testing"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/bootstrap"
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
			desc:        "valid request",
			token:       "token",
			externalID:  "external-id",
			externalKey: "external-key",
			err:         nil,
		},
		{
			desc:        "empty token",
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
			err:         nil,
		},
		{
			desc:        "empty external key and external ID",
			token:       "token",
			externalID:  "",
			externalKey: "",
			err:         apiutil.ErrMissingID,
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
		desc     string
		configID string
		err      error
	}{
		{
			desc:     "empty config id",
			configID: "",
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			configID: tc.configID,
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

func TestDeviceBootstrapReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		req  deviceBootstrapReq
		err  error
	}{
		{
			desc: "valid proof request",
			req: deviceBootstrapReq{
				externalID: "external-id",
				DeviceBootstrapProof: bootstrap.DeviceBootstrapProof{
					ChallengeID: "challenge-id",
					DeviceNonce: "device-nonce",
					Proof:       "proof",
				},
			},
		},
		{
			desc: "empty external id",
			req: deviceBootstrapReq{
				DeviceBootstrapProof: bootstrap.DeviceBootstrapProof{
					ChallengeID: "challenge-id",
					DeviceNonce: "device-nonce",
					Proof:       "proof",
				},
			},
			err: apiutil.ErrMissingID,
		},
		{
			desc: "empty proof",
			req: deviceBootstrapReq{
				externalID: "external-id",
				DeviceBootstrapProof: bootstrap.DeviceBootstrapProof{
					ChallengeID: "challenge-id",
					DeviceNonce: "device-nonce",
				},
			},
			err: apiutil.ErrMalformedRequestBody,
		},
	}

	for _, tc := range cases {
		err := tc.req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChangeConfigStatusReqValidation(t *testing.T) {
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
		{
			desc:  "valid request",
			token: "token",
			id:    "id",
			err:   nil,
		},
	}

	for _, tc := range cases {
		req := changeConfigStatusReq{
			token: tc.token,
			id:    tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
