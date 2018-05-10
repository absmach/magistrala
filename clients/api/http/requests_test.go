package http

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/clients"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

const wrong string = "?"

var (
	client  = clients.Client{Type: "app"}
	channel = clients.Channel{}
)

func TestIdentityReqValidation(t *testing.T) {
	cases := map[string]struct {
		key string
		err error
	}{
		"non-empty token": {uuid.NewV4().String(), nil},
		"empty token":     {"", clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		req := identityReq{tc.key}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestAddClientReqValidation(t *testing.T) {
	key := uuid.NewV4().String()

	cases := map[string]struct {
		client clients.Client
		key    string
		err    error
	}{
		"valid client addition request": {client, key, nil},
		"missing token":                 {client, "", clients.ErrUnauthorizedAccess},
		"wrong client type":             {clients.Client{Type: wrong}, key, clients.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		req := addClientReq{
			key:    tc.key,
			client: tc.client,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateClientReqValidation(t *testing.T) {
	key := uuid.NewV4().String()
	id := uuid.NewV4().String()

	cases := map[string]struct {
		client clients.Client
		id     string
		key    string
		err    error
	}{
		"valid client update request": {client, id, key, nil},
		"non-uuid client ID":          {client, wrong, key, clients.ErrNotFound},
		"missing token":               {client, id, "", clients.ErrUnauthorizedAccess},
		"wrong client type":           {clients.Client{Type: "invalid"}, id, key, clients.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		req := updateClientReq{
			key:    tc.key,
			id:     tc.id,
			client: tc.client,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannelReqValidation(t *testing.T) {
	key := uuid.NewV4().String()

	cases := map[string]struct {
		channel clients.Channel
		key     string
		err     error
	}{
		"valid channel creation request": {channel, key, nil},
		"missing token":                  {channel, "", clients.ErrUnauthorizedAccess},
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
		channel clients.Channel
		id      string
		key     string
		err     error
	}{
		"valid channel update request": {channel, id, key, nil},
		"non-uuid channel ID":          {channel, wrong, key, clients.ErrNotFound},
		"missing token":                {channel, id, "", clients.ErrUnauthorizedAccess},
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
		"missing token":                  {id, "", clients.ErrUnauthorizedAccess},
		"non-uuid resource ID":           {wrong, key, clients.ErrNotFound},
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
		"missing token":         {"", value, value, clients.ErrUnauthorizedAccess},
		"negative offset":       {key, -value, value, clients.ErrMalformedEntity},
		"zero limit":            {key, value, 0, clients.ErrMalformedEntity},
		"negative limit":        {key, value, -value, clients.ErrMalformedEntity},
		"too big limit":         {key, value, 20 * value, clients.ErrMalformedEntity},
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
