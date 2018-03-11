package manager_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/mocks"
	"github.com/stretchr/testify/assert"
)

const wrong string = "wrong-value"

var (
	user    manager.User    = manager.User{"user@example.com", "password"}
	client  manager.Client  = manager.Client{ID: "1", Type: "app", Name: "test", Key: "1"}
	channel manager.Channel = manager.Channel{ID: "1", Name: "test", Clients: []manager.Client{client}}
)

func newService() manager.Service {
	users := mocks.NewUserRepository()
	clients := mocks.NewClientRepository()
	channels := mocks.NewChannelRepository()
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()

	return manager.New(users, clients, channels, hasher, idp)
}

func TestRegister(t *testing.T) {
	svc := newService()

	cases := map[string]struct {
		user manager.User
		err  error
	}{
		"register new user":      {user, nil},
		"register existing user": {user, manager.ErrConflict},
	}

	for desc, tc := range cases {
		err := svc.Register(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestLogin(t *testing.T) {
	svc := newService()
	svc.Register(user)

	cases := map[string]struct {
		user manager.User
		err  error
	}{
		"login with good credentials": {user, nil},
		"login with wrong e-mail":     {manager.User{wrong, user.Password}, manager.ErrUnauthorizedAccess},
		"login with wrong password":   {manager.User{user.Email, wrong}, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.Login(tc.user)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestAddClient(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	cases := map[string]struct {
		client manager.Client
		key    string
		err    error
	}{
		"add new app":                       {manager.Client{Type: "app", Name: "a"}, key, nil},
		"add new device":                    {manager.Client{Type: "device", Name: "b"}, key, nil},
		"add client with wrong credentials": {manager.Client{Type: "app", Name: "d"}, wrong, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.AddClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateClient(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.AddClient(key, client)

	cases := map[string]struct {
		client manager.Client
		key    string
		err    error
	}{
		"update existing client":               {client, key, nil},
		"update client with wrong credentials": {client, wrong, manager.ErrUnauthorizedAccess},
		"update non-existing client":           {manager.Client{ID: "2", Type: "app", Name: "d"}, key, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewClient(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.AddClient(key, client)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing client":               {client.ID, key, nil},
		"view client with wrong credentials": {client.ID, wrong, manager.ErrUnauthorizedAccess},
		"view non-existing client":           {wrong, key, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListClients(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	cases := map[string]struct {
		key string
		err error
	}{
		"list clients":                        {key, nil},
		"list clients with wrong credentials": {wrong, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.ListClients(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveClient(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.AddClient(key, client)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove client with wrong credentials": {client.ID, "?", manager.ErrUnauthorizedAccess},
		"remove existing client":               {client.ID, key, nil},
		"remove removed client":                {client.ID, key, nil},
		"remove non-existing client":           {"?", key, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	cases := map[string]struct {
		channel manager.Channel
		key     string
		err     error
	}{
		"create channel":                        {manager.Channel{}, key, nil},
		"create channel with wrong credentials": {manager.Channel{}, wrong, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.CreateChannel(key, channel)

	cases := map[string]struct {
		channel manager.Channel
		key     string
		err     error
	}{
		"update existing channel":               {channel, key, nil},
		"update channel with wrong credentials": {channel, wrong, manager.ErrUnauthorizedAccess},
		"update non-existing channel":           {manager.Channel{ID: "2", Name: "test"}, key, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.CreateChannel(key, channel)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing channel":               {channel.ID, key, nil},
		"view channel with wrong credentials": {channel.ID, wrong, manager.ErrUnauthorizedAccess},
		"view non-existing channel":           {wrong, key, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	cases := map[string]struct {
		key string
		err error
	}{
		"list channels":                        {key, nil},
		"list channels with wrong credentials": {wrong, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.ListChannels(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)
	svc.CreateChannel(key, channel)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove channel with wrong credentials": {channel.ID, wrong, manager.ErrUnauthorizedAccess},
		"remove existing channel":               {channel.ID, key, nil},
		"remove removed channel":                {channel.ID, key, nil},
		"remove non-existing channel":           {channel.ID, key, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	clientId, _ := svc.AddClient(key, client)
	chanId, _ := svc.CreateChannel(key, channel)

	cases := map[string]struct {
		key      string
		chanId   string
		clientId string
		err      error
	}{
		"connect client":                         {key, chanId, clientId, nil},
		"connect client with wrong credentials":  {wrong, chanId, clientId, manager.ErrUnauthorizedAccess},
		"connect client to non-existing channel": {key, wrong, clientId, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.Connect(tc.key, tc.chanId, tc.clientId)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	clientId, _ := svc.AddClient(key, client)
	chanId, _ := svc.CreateChannel(key, channel)

	svc.Connect(key, chanId, clientId)

	cases := map[string]struct {
		key      string
		chanId   string
		clientId string
		err      error
	}{
		"disconnect connected client":                 {key, chanId, clientId, nil},
		"disconnect disconnected client":              {key, chanId, clientId, manager.ErrNotFound},
		"disconnect client with wrong credentials":    {wrong, chanId, clientId, manager.ErrUnauthorizedAccess},
		"disconnect client from non-existing channel": {key, wrong, clientId, manager.ErrNotFound},
		"disconnect non-existing client":              {key, chanId, wrong, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.Disconnect(tc.key, tc.chanId, tc.clientId)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}

}

func TestIdentity(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	cases := map[string]struct {
		key string
		err error
	}{
		"valid token's identity":   {key, nil},
		"invalid token's identity": {"", manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.Identity(tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCanAccess(t *testing.T) {
	svc := newService()
	svc.Register(user)
	key, _ := svc.Login(user)

	svc.AddClient(key, client)
	svc.CreateChannel(key, channel)

	cases := map[string]struct {
		key     string
		channel string
		err     error
	}{
		"allowed access":              {client.Key, channel.ID, nil},
		"not-connected cannot access": {wrong, channel.ID, manager.ErrUnauthorizedAccess},
		"access non-existing channel": {client.Key, wrong, manager.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccess(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
