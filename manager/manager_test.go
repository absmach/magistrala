package manager_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/mocks"
	"github.com/stretchr/testify/assert"
)

var (
	users    manager.UserRepository    = mocks.NewUserRepository()
	clients  manager.ClientRepository  = mocks.NewClientRepository()
	channels manager.ChannelRepository = mocks.NewChannelRepository()
	hasher   manager.Hasher            = mocks.NewHasher()
	idp      manager.IdentityProvider  = mocks.NewIdentityProvider()
	svc      manager.Service           = manager.NewService(users, clients, channels, hasher, idp)
)

func TestRegister(t *testing.T) {
	cases := []struct {
		user manager.User
		err  error
	}{
		{manager.User{"foo@bar.com", "pass"}, nil},
		{manager.User{"foo@bar.com", "pass"}, manager.ErrConflict},
	}

	for i, tc := range cases {
		e := svc.Register(tc.user)
		assert.Equal(t, tc.err, e, fmt.Sprintf("failed %d\n", i))
	}
}

func TestLogin(t *testing.T) {
	cases := []struct {
		user manager.User
		key  string
		err  error
	}{
		{manager.User{"foo@bar.com", "pass"}, "foo@bar.com", nil},
		{manager.User{"new@bar.com", "pass"}, "", manager.ErrUnauthorizedAccess},
		{manager.User{"foo@bar.com", ""}, "", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		k, e := svc.Login(tc.user)
		assert.Equal(t, tc.key, k, fmt.Sprintf("bad key at %d\n", i))
		assert.Equal(t, tc.err, e, fmt.Sprintf("failed %d\n", i))
	}
}

func TestIdentity(t *testing.T) {
	cases := []struct {
		key string
		id  string
		err error
	}{
		{"foo@bar.com", "foo@bar.com", nil},
		{"", "", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		id, err := svc.Identity(tc.key)
		assert.Equal(t, tc.id, id, fmt.Sprintf("unexpected id at %d\n", i))
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestAddClient(t *testing.T) {
	cases := []struct {
		key    string
		client manager.Client
		id     string
		err    error
	}{
		{"foo@bar.com", manager.Client{Type: "app", Name: "a"}, "1", nil},
		{"foo@bar.com", manager.Client{Type: "device", Name: "b"}, "2", nil},
		{"", manager.Client{Type: "app", Name: "d"}, "", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		id, err := svc.AddClient(tc.key, tc.client)
		assert.Equal(t, tc.id, id, fmt.Sprintf("unexpected id at %d\n", i))
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestUpdateClient(t *testing.T) {
	cases := []struct {
		key    string
		client manager.Client
		err    error
	}{
		{"foo@bar.com", manager.Client{ID: "1", Type: "app", Name: "aa"}, nil},
		{"foo@bar.com", manager.Client{ID: "2", Type: "device", Name: "bb"}, nil},
		{"", manager.Client{ID: "2", Type: "app", Name: "cc"}, manager.ErrUnauthorizedAccess},
		{"foo@bar.com", manager.Client{ID: "3", Type: "app", Name: "d"}, manager.ErrNotFound},
	}

	for i, tc := range cases {
		err := svc.UpdateClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestViewClient(t *testing.T) {
	cases := []struct {
		id  string
		key string
		err error
	}{
		{"1", "foo@bar.com", nil},
		{"1", "", manager.ErrUnauthorizedAccess},
		{"5", "foo@bar.com", manager.ErrNotFound},
	}

	for i, tc := range cases {
		_, err := svc.ViewClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestListClients(t *testing.T) {
	cases := []struct {
		key string
		err error
	}{
		{"foo@bar.com", nil},
		{"", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		_, err := svc.ListClients(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestRemoveClient(t *testing.T) {
	cases := []struct {
		id  string
		key string
		err error
	}{
		{"1", "", manager.ErrUnauthorizedAccess},
		{"1", "foo@bar.com", nil},
		{"1", "foo@bar.com", nil},
		{"2", "foo@bar.com", nil},
	}

	for i, tc := range cases {
		err := svc.RemoveClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestCreateChannel(t *testing.T) {
	cases := []struct {
		key     string
		channel manager.Channel
		id      string
		err     error
	}{
		{"foo@bar.com", manager.Channel{Connected: []string{"1", "2"}}, "1", nil},
		{"foo@bar.com", manager.Channel{Connected: []string{"2"}}, "2", nil},
		{"", manager.Channel{Connected: []string{"1"}}, "", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		id, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.id, id, fmt.Sprintf("unexpected id at %d\n", i))
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestUpdateChannel(t *testing.T) {
	cases := []struct {
		key     string
		channel manager.Channel
		err     error
	}{
		{"foo@bar.com", manager.Channel{ID: "1", Connected: []string{"1"}}, nil},
		{"foo@bar.com", manager.Channel{ID: "2", Connected: []string{}}, nil},
		{"", manager.Channel{ID: "2", Connected: []string{"1"}}, manager.ErrUnauthorizedAccess},
		{"foo@bar.com", manager.Channel{ID: "3", Connected: []string{"1"}}, manager.ErrNotFound},
	}

	for i, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestViewChannel(t *testing.T) {
	cases := []struct {
		id  string
		key string
		err error
	}{
		{"1", "foo@bar.com", nil},
		{"1", "", manager.ErrUnauthorizedAccess},
		{"5", "foo@bar.com", manager.ErrNotFound},
	}

	for i, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestListChannels(t *testing.T) {
	cases := []struct {
		key string
		err error
	}{
		{"foo@bar.com", nil},
		{"", manager.ErrUnauthorizedAccess},
	}

	for i, tc := range cases {
		_, err := svc.ListChannels(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestCanAccess(t *testing.T) {
	cases := []struct {
		client  string
		channel string
		allowed bool
	}{
		{"1", "1", true},
		{"1", "2", false},
		{"", "1", false},
	}

	for i, tc := range cases {
		allowed := svc.CanAccess(tc.client, tc.channel)
		assert.Equal(t, tc.allowed, allowed, fmt.Sprintf("failed at %d\n", i))
	}
}

func TestRemoveChannel(t *testing.T) {
	cases := []struct {
		id  string
		key string
		err error
	}{
		{"1", "", manager.ErrUnauthorizedAccess},
		{"1", "foo@bar.com", nil},
		{"1", "foo@bar.com", nil},
		{"2", "foo@bar.com", nil},
	}

	for i, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("failed at %d\n", i))
	}
}
