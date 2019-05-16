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

	"github.com/gofrs/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddThingReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	valid := things.Thing{}

	cases := map[string]struct {
		thing things.Thing
		token string
		err   error
	}{
		"valid thing addition request": {
			thing: valid,
			token: token,
			err:   nil,
		},
		"missing token": {
			thing: valid,
			token: "",
			err:   things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		req := addThingReq{
			token:    tc.token,
			Name:     tc.thing.Name,
			Metadata: tc.thing.Metadata,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThingReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	valid := things.Thing{ID: "1"}

	cases := map[string]struct {
		thing things.Thing
		id    string
		token string
		err   error
	}{
		"valid thing update request": {
			thing: valid,
			id:    valid.ID,
			token: token,
			err:   nil,
		},
		"missing token": {
			thing: valid,
			id:    valid.ID,
			token: "",
			err:   things.ErrUnauthorizedAccess,
		},
		"empty thing id": {
			thing: valid,
			id:    "",
			token: token,
			err:   things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := updateThingReq{
			token:    tc.token,
			id:       tc.id,
			Name:     tc.thing.Name,
			Metadata: tc.thing.Metadata,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateKeyReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	thing := things.Thing{ID: "1", Key: "key"}

	cases := map[string]struct {
		token string
		id    string
		key   string
		err   error
	}{
		"valid key update request": {
			token: token,
			id:    thing.ID,
			key:   thing.Key,
			err:   nil,
		},
		"missing token": {
			token: "",
			id:    thing.ID,
			key:   thing.Key,
			err:   things.ErrUnauthorizedAccess,
		},
		"empty thing id": {
			token: token,
			id:    "",
			key:   thing.Key,
			err:   things.ErrMalformedEntity,
		},
		"empty key": {
			token: token,
			id:    thing.ID,
			key:   "",
			err:   things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := updateKeyReq{
			token: tc.token,
			id:    tc.id,
			Key:   tc.key,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannelReqValidation(t *testing.T) {
	channel := things.Channel{}
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	cases := map[string]struct {
		channel things.Channel
		token   string
		err     error
	}{
		"valid channel creation request": {
			channel: channel,
			token:   token,
			err:     nil,
		},
		"missing token": {
			channel: channel,
			token:   "",
			err:     things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		req := createChannelReq{
			token: tc.token,
			Name:  tc.channel.Name,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannelReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	channel := things.Channel{ID: "1"}

	cases := map[string]struct {
		channel things.Channel
		id      string
		token   string
		err     error
	}{
		"valid channel update request": {
			channel: channel,
			id:      channel.ID,
			token:   token,
			err:     nil,
		},
		"missing token": {
			channel: channel,
			id:      channel.ID,
			token:   "",
			err:     things.ErrUnauthorizedAccess,
		},
		"empty channel id": {
			channel: channel,
			id:      "",
			token:   token,
			err:     things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := updateChannelReq{
			token: tc.token,
			id:    tc.id,
			Name:  tc.channel.Name,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewResourceReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	id := uint64(1)

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"valid resource viewing request": {
			id:    strconv.FormatUint(id, 10),
			token: token,
			err:   nil,
		},
		"missing token": {
			id:    strconv.FormatUint(id, 10),
			token: "",
			err:   things.ErrUnauthorizedAccess,
		},
		"empty resource id": {
			id:    "",
			token: token,
			err:   things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := viewResourceReq{tc.token, tc.id}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListResourcesReqValidation(t *testing.T) {
	uuidToken, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	token := uuidToken.String()

	value := uint64(10)

	cases := map[string]struct {
		token  string
		offset uint64
		limit  uint64
		err    error
	}{
		"valid listing request": {
			token:  token,
			offset: value,
			limit:  value,
			err:    nil,
		},
		"missing token": {
			token:  "",
			offset: value,
			limit:  value,
			err:    things.ErrUnauthorizedAccess,
		},
		"zero limit": {
			token:  token,
			offset: value,
			limit:  0,
			err:    things.ErrMalformedEntity,
		},
		"too big limit": {
			token:  token,
			offset: value,
			limit:  20 * value,
			err:    things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := listResourcesReq{
			token:  tc.token,
			offset: tc.offset,
			limit:  tc.limit,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnectionReqValidation(t *testing.T) {
	cases := map[string]struct {
		token   string
		chanID  string
		thingID string
		err     error
	}{
		"valid token": {
			token:   "valid-token",
			chanID:  "1",
			thingID: "1",
			err:     nil,
		},
		"empty token": {
			token:   "",
			chanID:  "1",
			thingID: "1",
			err:     things.ErrUnauthorizedAccess,
		},
		"empty channel id": {
			token:   "valid-token",
			chanID:  "",
			thingID: "1",
			err:     things.ErrMalformedEntity,
		},
		"empty thing id": {
			token:   "valid-token",
			chanID:  "1",
			thingID: "",
			err:     things.ErrMalformedEntity,
		},
	}

	for desc, tc := range cases {
		req := connectionReq{
			token:   tc.token,
			chanID:  tc.chanID,
			thingID: tc.thingID,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
