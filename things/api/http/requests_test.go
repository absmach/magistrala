//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"fmt"
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
		"non-empty token": {key: uuid.NewV4().String(), err: nil},
		"empty token":     {key: "", err: things.ErrUnauthorizedAccess},
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
	invalid := things.Thing{Type: "?"}

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"valid thing addition request": {thing: valid, key: key, err: nil},
		"missing token":                {thing: valid, key: "", err: things.ErrUnauthorizedAccess},
		"wrong thing type":             {thing: invalid, key: key, err: things.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		req := addThingReq{
			key:   tc.key,
			thing: tc.thing,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThingReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	valid := things.Thing{ID: 1, Type: "app"}
	invalid := things.Thing{ID: 0, Type: "?"}

	cases := map[string]struct {
		thing things.Thing
		id    uint64
		key   string
		err   error
	}{
		"valid thing update request": {thing: valid, id: valid.ID, key: key, err: nil},
		"invalid thing ID":           {thing: valid, id: invalid.ID, key: key, err: things.ErrNotFound},
		"missing token":              {thing: valid, id: valid.ID, key: "", err: things.ErrUnauthorizedAccess},
		"wrong thing type":           {thing: invalid, id: valid.ID, key: key, err: things.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		req := updateThingReq{
			key:   tc.key,
			id:    tc.id,
			thing: tc.thing,
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
		"valid channel creation request": {channel: channel, key: key, err: nil},
		"missing token":                  {channel: channel, key: "", err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		req := createChannelReq{
			key:     tc.key,
			channel: tc.channel,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannelReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	channel := things.Channel{ID: 1}
	wrongID := uint64(0)

	cases := map[string]struct {
		channel things.Channel
		id      uint64
		key     string
		err     error
	}{
		"valid channel update request": {channel: channel, id: channel.ID, key: key, err: nil},
		"invalid channel ID":           {channel: channel, id: wrongID, key: key, err: things.ErrNotFound},
		"missing token":                {channel: channel, id: channel.ID, key: "", err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		req := updateChannelReq{
			key:     tc.key,
			id:      tc.id,
			channel: tc.channel,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewResourceReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	id := uint64(1)
	wrongID := uint64(0)

	cases := map[string]struct {
		id  uint64
		key string
		err error
	}{
		"valid resource viewing request": {id: id, key: key, err: nil},
		"missing token":                  {id: id, key: "", err: things.ErrUnauthorizedAccess},
		"invalid resource ID":            {id: wrongID, key: key, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		req := viewResourceReq{tc.key, tc.id}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListResourcesReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	value := 10

	cases := map[string]struct {
		key    string
		offset int
		limit  int
		err    error
	}{
		"valid listing request": {key: key, offset: value, limit: value, err: nil},
		"missing token":         {key: "", offset: value, limit: value, err: things.ErrUnauthorizedAccess},
		"negative offset":       {key: key, offset: -value, limit: value, err: things.ErrMalformedEntity},
		"zero limit":            {key: key, offset: value, limit: 0, err: things.ErrMalformedEntity},
		"negative limit":        {key: key, offset: value, limit: -value, err: things.ErrMalformedEntity},
		"too big limit":         {key: key, offset: value, limit: 20 * value, err: things.ErrMalformedEntity},
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
