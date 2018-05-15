package http

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/things"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

const wrong string = "?"

var (
	thing   = things.Thing{Type: "app"}
	channel = things.Channel{}
)

func TestIdentityReqValidation(t *testing.T) {
	cases := map[string]struct {
		key string
		err error
	}{
		"non-empty token": {uuid.NewV4().String(), nil},
		"empty token":     {"", things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		req := identityReq{tc.key}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestAddThingReqValidation(t *testing.T) {
	key := uuid.NewV4().String()

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"valid thing addition request": {thing, key, nil},
		"missing token":                {thing, "", things.ErrUnauthorizedAccess},
		"wrong thing type":             {things.Thing{Type: wrong}, key, things.ErrMalformedEntity},
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
	id := uuid.NewV4().String()

	cases := map[string]struct {
		thing things.Thing
		id    string
		key   string
		err   error
	}{
		"valid thing update request": {thing, id, key, nil},
		"non-uuid thing ID":          {thing, wrong, key, things.ErrNotFound},
		"missing token":              {thing, id, "", things.ErrUnauthorizedAccess},
		"wrong thing type":           {things.Thing{Type: "invalid"}, id, key, things.ErrMalformedEntity},
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
	key := uuid.NewV4().String()

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"valid channel creation request": {channel, key, nil},
		"missing token":                  {channel, "", things.ErrUnauthorizedAccess},
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
	id := uuid.NewV4().String()

	cases := map[string]struct {
		channel things.Channel
		id      string
		key     string
		err     error
	}{
		"valid channel update request": {channel, id, key, nil},
		"non-uuid channel ID":          {channel, wrong, key, things.ErrNotFound},
		"missing token":                {channel, id, "", things.ErrUnauthorizedAccess},
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
	id := uuid.NewV4().String()

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"valid resource viewing request": {id, key, nil},
		"missing token":                  {id, "", things.ErrUnauthorizedAccess},
		"non-uuid resource ID":           {wrong, key, things.ErrNotFound},
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
		"valid listing request": {key, value, value, nil},
		"missing token":         {"", value, value, things.ErrUnauthorizedAccess},
		"negative offset":       {key, -value, value, things.ErrMalformedEntity},
		"zero limit":            {key, value, 0, things.ErrMalformedEntity},
		"negative limit":        {key, value, -value, things.ErrMalformedEntity},
		"too big limit":         {key, value, 20 * value, things.ErrMalformedEntity},
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
