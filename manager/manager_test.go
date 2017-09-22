package manager_test

import (
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
	var cases = []struct {
		user manager.User
		err  error
	}{
		{manager.User{"foo@bar.com", "pass"}, nil},
		{manager.User{"foo@bar.com", "pass"}, manager.ErrConflict},
		{manager.User{"", "pass"}, manager.ErrInvalidCredentials},
		{manager.User{"abc@bar.com", ""}, manager.ErrInvalidCredentials},
		{manager.User{"abc@bar.com", "pass"}, nil},
	}

	for _, tc := range cases {
		e := svc.Register(tc.user)
		assert.Equal(t, tc.err, e, "unexpected error occurred")
	}
}

func TestLogin(t *testing.T) {
	var cases = []struct {
		user manager.User
		key  string
		err  error
	}{
		{manager.User{"foo@bar.com", "pass"}, "foo@bar.com", nil},
		{manager.User{"new@bar.com", "pass"}, "", manager.ErrInvalidCredentials},
		{manager.User{"foo@bar.com", ""}, "", manager.ErrInvalidCredentials},
	}

	for _, tc := range cases {
		k, e := svc.Login(tc.user)
		assert.Equal(t, tc.key, k, "unexpected key retrieved")
		assert.Equal(t, tc.err, e, "unexpected error occurred")
	}
}

func TestAddClient(t *testing.T) {
	var cases = []struct {
		key    string
		client manager.Client
		id     string
		err    error
	}{
		{"foo@bar.com", manager.Client{Type: "app", Name: "a"}, "1", nil},
		{"foo@bar.com", manager.Client{Type: "device", Name: "b"}, "2", nil},
		{"", manager.Client{Type: "app", Name: "d"}, "", manager.ErrUnauthorizedAccess},
		{"foo@bar.com", manager.Client{Type: "invalid", Name: "d"}, "", manager.ErrMalformedClient},
	}

	for _, tc := range cases {
		id, err := svc.AddClient(tc.key, tc.client)
		assert.Equal(t, tc.id, id, "unexpected id retrieved")
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestUpdateClient(t *testing.T) {
	var cases = []struct {
		key    string
		client manager.Client
		err    error
	}{
		{"foo@bar.com", manager.Client{ID: "1", Type: "app", Name: "aa"}, nil},
		{"foo@bar.com", manager.Client{ID: "2", Type: "device", Name: "bb"}, nil},
		{"", manager.Client{ID: "2", Type: "app", Name: "cc"}, manager.ErrUnauthorizedAccess},
		{"foo@bar.com", manager.Client{ID: "2", Type: "invalid", Name: "d"}, manager.ErrMalformedClient},
		{"foo@bar.com", manager.Client{ID: "3", Type: "app", Name: "d"}, manager.ErrNotFound},
	}

	for _, tc := range cases {
		err := svc.UpdateClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestViewClient(t *testing.T) {
	var cases = []struct {
		id  string
		key string
		err error
	}{
		{"1", "foo@bar.com", nil},
		{"1", "", manager.ErrUnauthorizedAccess},
		{"5", "foo@bar.com", manager.ErrNotFound},
	}

	for _, tc := range cases {
		_, err := svc.ViewClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestListClients(t *testing.T) {
	var cases = []struct {
		key string
		err error
	}{
		{"foo@bar.com", nil},
		{"", manager.ErrUnauthorizedAccess},
	}

	for _, tc := range cases {
		_, err := svc.ListClients(tc.key)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestRemoveClient(t *testing.T) {
	var cases = []struct {
		id  string
		key string
		err error
	}{
		{"1", "", manager.ErrUnauthorizedAccess},
		{"1", "foo@bar.com", nil},
		{"1", "foo@bar.com", nil},
		{"2", "foo@bar.com", nil},
	}

	for _, tc := range cases {
		err := svc.RemoveClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestCreateChannel(t *testing.T) {
	var cases = []struct {
		key     string
		channel manager.Channel
		id      string
		err     error
	}{
		{"foo@bar.com", manager.Channel{Connected: []string{"1", "2"}}, "1", nil},
		{"foo@bar.com", manager.Channel{Connected: []string{"2"}}, "2", nil},
		{"", manager.Channel{Connected: []string{"1"}}, "", manager.ErrUnauthorizedAccess},
	}

	for _, tc := range cases {
		id, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.id, id, "unexpected id retrieved")
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestUpdateChannel(t *testing.T) {
	var cases = []struct {
		key     string
		channel manager.Channel
		err     error
	}{
		{"foo@bar.com", manager.Channel{ID: "1", Connected: []string{"1"}}, nil},
		{"foo@bar.com", manager.Channel{ID: "2", Connected: []string{}}, nil},
		{"", manager.Channel{ID: "2", Connected: []string{"1"}}, manager.ErrUnauthorizedAccess},
		{"foo@bar.com", manager.Channel{ID: "3", Connected: []string{"1"}}, manager.ErrNotFound},
	}

	for _, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestViewChannel(t *testing.T) {
	var cases = []struct {
		id  string
		key string
		err error
	}{
		{"1", "foo@bar.com", nil},
		{"1", "", manager.ErrUnauthorizedAccess},
		{"5", "foo@bar.com", manager.ErrNotFound},
	}

	for _, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestListChannels(t *testing.T) {
	var cases = []struct {
		key string
		err error
	}{
		{"foo@bar.com", nil},
		{"", manager.ErrUnauthorizedAccess},
	}

	for _, tc := range cases {
		_, err := svc.ListChannels(tc.key)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}

func TestCanAccess(t *testing.T) {
	var cases = []struct {
		client  string
		channel string
		allowed bool
	}{
		{"1", "1", true},
		{"1", "2", false},
		{"", "1", false},
	}

	for _, tc := range cases {
		allowed := svc.CanAccess(tc.client, tc.channel)
		assert.Equal(t, tc.allowed, allowed, "unexpected value occurred")
	}
}

func TestRemoveChannel(t *testing.T) {
	var cases = []struct {
		id  string
		key string
		err error
	}{
		{"1", "", manager.ErrUnauthorizedAccess},
		{"1", "foo@bar.com", nil},
		{"1", "foo@bar.com", nil},
		{"2", "foo@bar.com", nil},
	}

	for _, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, "unexpected error occurred")
	}
}
