package api

import (
	"fmt"
	"testing"

	"github.com/gocql/gocql"
	"github.com/mainflux/mainflux/manager"
	"github.com/stretchr/testify/assert"
)

func TestUserReqValidation(t *testing.T) {
	cases := []struct {
		user manager.User
		err  error
	}{
		{manager.User{"foo@example.com", "pass"}, nil},
		{manager.User{"invalid", "pass"}, manager.ErrMalformedEntity},
		{manager.User{"", "pass"}, manager.ErrMalformedEntity},
		{manager.User{"foo@example.com", ""}, manager.ErrMalformedEntity},
	}

	for i, tc := range cases {
		req := userReq{tc.user}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestIdentityReqValidation(t *testing.T) {
	cases := []struct {
		key string
		err error
	}{
		{"valid", nil},
		{"", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		req := identityReq{tc.key}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestAddClientReqValidation(t *testing.T) {
	key := "key"
	vc := manager.Client{Type: "app"}

	cases := []struct {
		key    string
		client manager.Client
		err    error
	}{
		{key, vc, nil},
		{"", vc, manager.ErrUnauthorizedAccess},
		{key, manager.Client{Type: "invalid"}, manager.ErrMalformedEntity},
	}

	for i, tc := range cases {
		req := addClientReq{
			key:    tc.key,
			client: tc.client,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestUpdateClientReqValidation(t *testing.T) {
	key := "key"
	uuid := gocql.TimeUUID().String()
	vc := manager.Client{Type: "app"}

	cases := []struct {
		key    string
		id     string
		client manager.Client
		err    error
	}{
		{key, uuid, vc, nil},
		{key, "non-uuid", vc, manager.ErrNotFound},
		{"", uuid, vc, manager.ErrUnauthorizedAccess},
		{key, uuid, manager.Client{Type: "invalid"}, manager.ErrMalformedEntity},
	}

	for i, tc := range cases {
		req := updateClientReq{
			key:    tc.key,
			id:     tc.id,
			client: tc.client,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestCreateChannelReqValidation(t *testing.T) {
	key := "key"
	vc := manager.Channel{}

	cases := []struct {
		key     string
		channel manager.Channel
		err     error
	}{
		{key, vc, nil},
		{"", vc, manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		req := createChannelReq{
			key:     tc.key,
			channel: tc.channel,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestUpdateChannelReqValidation(t *testing.T) {
	key := "key"
	uuid := gocql.TimeUUID().String()
	vc := manager.Channel{}

	cases := []struct {
		key     string
		id      string
		channel manager.Channel
		err     error
	}{
		{key, uuid, vc, nil},
		{key, "non-uuid", vc, manager.ErrNotFound},
		{"", uuid, vc, manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		req := updateChannelReq{
			key:     tc.key,
			id:      tc.id,
			channel: tc.channel,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestViewResourceReqValidation(t *testing.T) {
	key := "key"
	uuid := gocql.TimeUUID().String()

	cases := []struct {
		key string
		id  string
		err error
	}{
		{key, uuid, nil},
		{"", uuid, manager.ErrUnauthorizedAccess},
		{key, "non-uuid", manager.ErrNotFound},
	}

	for i, tc := range cases {
		req := viewResourceReq{tc.key, tc.id}
		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestListResourcesReqValidation(t *testing.T) {
	cases := []struct {
		key    string
		size   int
		offset int
		err    error
	}{
		{"key", 10, 10, nil},
		{"", 10, 10, manager.ErrUnauthorizedAccess},
		{"key", 10, -10, manager.ErrMalformedEntity},
		{"key", 0, 10, manager.ErrMalformedEntity},
		{"key", -10, 10, manager.ErrMalformedEntity},
	}

	for i, tc := range cases {
		req := listResourcesReq{
			key:    tc.key,
			size:   tc.size,
			offset: tc.offset,
		}

		err := req.validate()
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}
