package api

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/stretchr/testify/assert"
)

func TestAddReqValidation(t *testing.T) {
	cases := []struct {
		desc        string
		key         string
		externalID  string
		externalKey string
		err         error
	}{
		{
			desc:        "empty key",
			key:         "",
			externalID:  "external-id",
			externalKey: "external-key",
			err:         bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:        "empty external ID",
			key:         "key",
			externalID:  "",
			externalKey: "external-key",
			err:         bootstrap.ErrMalformedEntity,
		},
		{
			desc:        "empty external key",
			key:         "key",
			externalID:  "external-id",
			externalKey: "",
			err:         bootstrap.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		req := addReq{
			key:         tc.key,
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
			err:  bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  bootstrap.ErrMalformedEntity,
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
			err:  bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  bootstrap.ErrMalformedEntity,
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
		desc     string
		key      string
		thingKey string
		err      error
	}{
		{
			desc:     "empty key",
			key:      "",
			thingKey: "thingKey",
			err:      bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:     "empty thing key",
			key:      "key",
			thingKey: "",
			err:      bootstrap.ErrNotFound,
		},
	}

	for _, tc := range cases {
		req := updateCertReq{
			key:      tc.key,
			thingKey: tc.thingKey,
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
			err:  bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc: "empty id",
			key:  "key",
			id:   "",
			err:  bootstrap.ErrMalformedEntity,
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
			err:    bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:   "too large limit",
			key:    "key",
			offset: 0,
			limit:  maxLimit + 1,
			err:    bootstrap.ErrMalformedEntity,
		},
		{
			desc:   "zero limit",
			key:    "key",
			offset: 0,
			limit:  0,
			err:    bootstrap.ErrMalformedEntity,
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
			err:       bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:      "empty external id",
			externKey: "key",
			externID:  "",
			err:       bootstrap.ErrMalformedEntity,
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
			err:   bootstrap.ErrUnauthorizedAccess,
		},
		{
			desc:  "empty id",
			key:   "key",
			id:    "",
			state: bootstrap.State(0),
			err:   bootstrap.ErrMalformedEntity,
		},
		{
			desc:  "invalid state",
			key:   "key",
			id:    "id",
			state: bootstrap.State(14),
			err:   bootstrap.ErrMalformedEntity,
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
