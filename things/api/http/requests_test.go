//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux/things"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestIdentityReqValidation(t *testing.T) {
	cases := map[string]struct {
		key string
		err error
	}{
		"non-empty token": {
			key: uuid.NewV4().String(),
			err: nil,
		},
		"empty token": {
			key: "",
			err: things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		req := identityReq{key: tc.key}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestAddThingReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	valid := things.Thing{Type: "app"}
	invalid := things.Thing{ID: "0", Type: ""}

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"valid thing addition request": {
			thing: valid,
			key:   key,
			err:   nil,
		},
		"missing token": {
			thing: valid,
			key:   "", err: things.ErrUnauthorizedAccess,
		},
		"empty thing type": {
			thing: invalid,
			key:   key,
			err:   things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := addThingReq{
			key:      tc.key,
			Name:     tc.thing.Name,
			Type:     tc.thing.Type,
			Metadata: tc.thing.Metadata,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThingReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	valid := things.Thing{ID: "1", Type: "app"}
	invalid := things.Thing{ID: "0", Type: ""}

	cases := map[string]struct {
		thing things.Thing
		id    string
		key   string
		err   error
	}{
		"valid thing update request": {
			thing: valid,
			id:    valid.ID,
			key:   key,
			err:   nil,
		},
		"missing token": {
			thing: valid,
			id:    valid.ID,
			key:   "",
			err:   things.ErrUnauthorizedAccess,
		},
		"empty thing type": {
			thing: invalid,
			id:    valid.ID,
			key:   key,
			err:   things.ErrMalformedEntity,
		},
		"empty thing id": {
			thing: valid,
			id:    "",
			key:   key,
			err:   things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := updateThingReq{
			key:      tc.key,
			id:       tc.id,
			Name:     tc.thing.Name,
			Type:     tc.thing.Type,
			Metadata: tc.thing.Metadata,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannelReqValidation(t *testing.T) {
	channel := things.Channel{}
	key := uuid.NewV4().String()

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"valid channel creation request": {
			channel: channel,
			key:     key,
			err:     nil,
		},
		"missing token": {
			channel: channel,
			key:     "",
			err:     things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		req := createChannelReq{
			key:  tc.key,
			Name: tc.channel.Name,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannelReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	channel := things.Channel{ID: "1"}

	cases := map[string]struct {
		channel things.Channel
		id      string
		key     string
		err     error
	}{
		"valid channel update request": {
			channel: channel,
			id:      channel.ID,
			key:     key,
			err:     nil,
		},
		"missing token": {
			channel: channel,
			id:      channel.ID,
			key:     "",
			err:     things.ErrUnauthorizedAccess,
		},
		"empty channel id": {
			channel: channel,
			id:      "",
			key:     key,
			err:     things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := updateChannelReq{
			key:  tc.key,
			id:   tc.id,
			Name: tc.channel.Name,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewResourceReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	id := uint64(1)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"valid resource viewing request": {
			id:  strconv.FormatUint(id, 10),
			key: key,
			err: nil,
		},
		"missing token": {
			id:  strconv.FormatUint(id, 10),
			key: "",
			err: things.ErrUnauthorizedAccess,
		},
		"empty resource id": {
			id:  "",
			key: key,
			err: things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := viewResourceReq{tc.key, tc.id}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListResourcesReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	value := uint64(10)

	cases := map[string]struct {
		key    string
		offset uint64
		limit  uint64
		err    error
	}{
		"valid listing request": {
			key:    key,
			offset: value,
			limit:  value,
			err:    nil,
		},
		"missing token": {
			key:    "",
			offset: value,
			limit:  value,
			err:    things.ErrUnauthorizedAccess,
		},
		"zero limit": {
			key:    key,
			offset: value,
			limit:  0,
			err:    things.ErrMalformedEntity,
		},
		"too big limit": {
			key:    key,
			offset: value,
			limit:  20 * value,
			err:    things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := listResourcesReq{
			key:    tc.key,
			offset: tc.offset,
			limit:  tc.limit,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnectionReqValidation(t *testing.T) {
	cases := map[string]struct {
		key     string
		chanID  string
		thingID string
		err     error
	}{
		"valid key": {
			key:     "valid-key",
			chanID:  "1",
			thingID: "1",
			err:     nil,
		},
		"empty key": {
			key:     "",
			chanID:  "1",
			thingID: "1",
			err:     things.ErrUnauthorizedAccess,
		},
		"empty channel id": {
			key:     "valid-key",
			chanID:  "",
			thingID: "1",
			err:     things.ErrMalformedEntity,
		},
		"empty thing id": {
			key:     "valid-key",
			chanID:  "1",
			thingID: "",
			err:     things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := connectionReq{
			key:     tc.key,
			chanID:  tc.chanID,
			thingID: tc.thingID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
