package api

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/manager"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

const wrong string = "?"

var (
	client  manager.Client  = manager.Client{Type: "app"}
	channel manager.Channel = manager.Channel{}
)

func TestUserReqValidation(t *testing.T) {
	cases := map[string]struct {
		user manager.User
		err  error
	}{
		"valid user request": {manager.User{"foo@example.com", "pass"}, nil},
		"malformed e-mail":   {manager.User{wrong, "pass"}, manager.ErrMalformedEntity},
		"empty e-mail":       {manager.User{"", "pass"}, manager.ErrMalformedEntity},
		"empty password":     {manager.User{"foo@example.com", ""}, manager.ErrMalformedEntity},
	}

	for desc, tc := range cases {
		req := userReq{tc.user}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentityReqValidation(t *testing.T) {
	cases := map[string]struct {
		key string
		err error
	}{
		"non-empty token": {uuid.NewV4().String(), nil},
		"empty token":     {"", manager.ErrUnauthorizedAccess},
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
		client manager.Client
		key    string
		err    error
	}{
		"valid client addition request": {client, key, nil},
		"missing token":                 {client, "", manager.ErrUnauthorizedAccess},
		"wrong client type":             {manager.Client{Type: wrong}, key, manager.ErrMalformedEntity},
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
		client manager.Client
		id     string
		key    string
		err    error
	}{
		"valid client update request": {client, id, key, nil},
		"non-uuid client ID":          {client, wrong, key, manager.ErrNotFound},
		"missing token":               {client, id, "", manager.ErrUnauthorizedAccess},
		"wrong client type":           {manager.Client{Type: "invalid"}, id, key, manager.ErrMalformedEntity},
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
		channel manager.Channel
		key     string
		err     error
	}{
		"valid channel creation request": {channel, key, nil},
		"missing token":                  {channel, "", manager.ErrUnauthorizedAccess},
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
		channel manager.Channel
		id      string
		key     string
		err     error
	}{
		"valid channel update request": {channel, id, key, nil},
		"non-uuid channel ID":          {channel, wrong, key, manager.ErrNotFound},
		"missing token":                {channel, id, "", manager.ErrUnauthorizedAccess},
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
		"missing token":                  {id, "", manager.ErrUnauthorizedAccess},
		"non-uuid resource ID":           {wrong, key, manager.ErrNotFound},
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
		"missing token":         {"", value, value, manager.ErrUnauthorizedAccess},
		"negative offset":       {key, -value, value, manager.ErrMalformedEntity},
		"zero size":             {key, value, 0, manager.ErrMalformedEntity},
		"negative size":         {key, value, -value, manager.ErrMalformedEntity},
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
