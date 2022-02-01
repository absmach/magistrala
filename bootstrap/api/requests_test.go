package api

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/pkg/errors"
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
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "empty external ID",
			token:       "token",
			externalID:  "",
			externalKey: "external-key",
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "empty external key",
			token:       "token",
			externalID:  "external-id",
			externalKey: "",
			err:         errors.ErrMalformedEntity,
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
		key  string
		id   string
		err  error
	}{
		{
			desc: "empty key",
			key:  "",
			id:   "id",
			err:  errors.ErrAuthentication,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := entityReq{
			key: tc.key,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		id   string
		err  error
	}{
		{
			desc: "empty key",
			key:  "",
			id:   "id",
			err:  errors.ErrAuthentication,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			key: tc.key,
			id:  tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCertReqValidation(t *testing.T) {
	cases := []struct {
		desc    string
		key     string
		thingID string
		err     error
	}{
		{
			desc:    "empty key",
			key:     "",
			thingID: "thingID",
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "empty thing key",
			key:     "key",
			thingID: "",
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			key:     tc.key,
			thingID: tc.thingID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnReqValidation(t *testing.T) {
	cases := []struct {
		desc string
		key  string
		id   string
		err  error
	}{
		{
			desc: "empty key",
			key:  "",
			id:   "id",
			err:  errors.ErrAuthentication,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := updateReq{
			key: tc.key,
			id:  tc.id,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListReqValidation(t *testing.T) {
	cases := []struct {
		desc   string
		offset uint64
		key    string
		limit  uint64
		err    error
	}{
		{
			desc:   "empty key",
			key:    "",
			offset: 0,
			limit:  1,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "too large limit",
			key:    "key",
			offset: 0,
			limit:  maxLimit + 1,
			err:    errors.ErrMalformedEntity,
		},
		{
			desc:   "zero limit",
			key:    "key",
			offset: 0,
			limit:  0,
			err:    errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := listReq{
			key:    tc.key,
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
			err:       errors.ErrAuthentication,
		},
		{
			desc:      "empty external id",
			externKey: "key",
			externID:  "",
			err:       errors.ErrMalformedEntity,
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
		key   string
		id    string
		state bootstrap.State
		err   error
	}{
		{
			desc:  "empty key",
			key:   "",
			id:    "id",
			state: bootstrap.State(1),
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "empty id",
			key:   "key",
			id:    "",
			state: bootstrap.State(0),
			err:   errors.ErrMalformedEntity,
		},
		{
			desc:  "invalid state",
			key:   "key",
			id:    "id",
			state: bootstrap.State(14),
			err:   errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := changeStateReq{
			key:   tc.key,
			id:    tc.id,
			State: tc.state,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
