package clients_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/clients/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	wrong = "wrong-value"
	email = "user@example.com"
	token = "token"
)

var (
	client  = clients.Client{Type: "app", Name: "test"}
	channel = clients.Channel{Name: "test", Clients: []clients.Client{}}
)

func newService(tokens map[string]string) clients.Service {
	users := mocks.NewUsersService(tokens)
	clientsRepo := mocks.NewClientRepository()
	channelsRepo := mocks.NewChannelRepository(clientsRepo)
	hasher := mocks.NewHasher()
	idp := mocks.NewIdentityProvider()

	return clients.New(users, clientsRepo, channelsRepo, hasher, idp)
}

func TestAddClient(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		client clients.Client
		key    string
		err    error
	}{
		"add new app":                       {clients.Client{Type: "app", Name: "a"}, token, nil},
		"add new device":                    {clients.Client{Type: "device", Name: "b"}, token, nil},
		"add client with wrong credentials": {clients.Client{Type: "app", Name: "d"}, wrong, clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.AddClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateClient(t *testing.T) {
	svc := newService(map[string]string{token: email})
	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID

	cases := map[string]struct {
		client clients.Client
		key    string
		err    error
	}{
		"update existing client":               {client, token, nil},
		"update client with wrong credentials": {client, wrong, clients.ErrUnauthorizedAccess},
		"update non-existing client":           {clients.Client{ID: "2", Type: "app", Name: "d"}, token, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateClient(tc.key, tc.client)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewClient(t *testing.T) {
	svc := newService(map[string]string{token: email})
	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing client":               {client.ID, token, nil},
		"view client with wrong credentials": {client.ID, wrong, clients.ErrUnauthorizedAccess},
		"view non-existing client":           {wrong, token, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListClients(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.AddClient(token, client)
	}
	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list clients":                        {token, 0, 5, 5, nil},
		"list clients 5-10":                   {token, 5, 10, 5, nil},
		"list last client":                    {token, 9, 10, 1, nil},
		"list empty response":                 {token, 11, 10, 0, nil},
		"list offset < 0":                     {token, -1, 10, 0, nil},
		"list limit < 0":                      {token, 1, -10, 0, nil},
		"list limit = 0":                      {token, 1, 0, 0, nil},
		"list clients with wrong credentials": {wrong, 0, 0, 0, clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		cl, err := svc.ListClients(tc.key, tc.offset, tc.limit)
		size := len(cl)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveClient(t *testing.T) {
	svc := newService(map[string]string{token: email})
	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove client with wrong credentials": {client.ID, "?", clients.ErrUnauthorizedAccess},
		"remove existing client":               {client.ID, token, nil},
		"remove removed client":                {client.ID, token, nil},
		"remove non-existing client":           {"?", token, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveClient(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		channel clients.Channel
		key     string
		err     error
	}{
		"create channel":                        {clients.Channel{}, token, nil},
		"create channel with wrong credentials": {clients.Channel{}, wrong, clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	cases := map[string]struct {
		channel clients.Channel
		key     string
		err     error
	}{
		"update existing channel":               {channel, token, nil},
		"update channel with wrong credentials": {channel, wrong, clients.ErrUnauthorizedAccess},
		"update non-existing channel":           {clients.Channel{ID: "2", Name: "test"}, token, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing channel":               {channel.ID, token, nil},
		"view channel with wrong credentials": {channel.ID, wrong, clients.ErrUnauthorizedAccess},
		"view non-existing channel":           {wrong, token, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.CreateChannel(token, channel)
	}
	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list first 5 channels":                {token, 0, 5, 5, nil},
		"list channels 5-10 channels":          {token, 5, 10, 5, nil},
		"list last channel":                    {token, 6, 10, 4, nil},
		"list offset < 0":                      {token, -1, 10, 0, nil},
		"list limit < 0":                       {token, 1, -10, 0, nil},
		"list limit = 0":                       {token, 1, 0, 0, nil},
		"list channels with wrong credentials": {wrong, 0, 0, 0, clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		ch, err := svc.ListChannels(tc.key, tc.offset, tc.limit)
		size := len(ch)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove channel with wrong credentials": {channel.ID, wrong, clients.ErrUnauthorizedAccess},
		"remove existing channel":               {channel.ID, token, nil},
		"remove removed channel":                {channel.ID, token, nil},
		"remove non-existing channel":           {channel.ID, token, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	cases := map[string]struct {
		key      string
		chanID   string
		clientID string
		err      error
	}{
		"connect client":                         {token, channel.ID, client.ID, nil},
		"connect client with wrong credentials":  {wrong, channel.ID, client.ID, clients.ErrUnauthorizedAccess},
		"connect client to non-existing channel": {token, wrong, client.ID, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.Connect(tc.key, tc.chanID, tc.clientID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	svc.Connect(token, chanID, clientID)

	cases := []struct {
		desc     string
		key      string
		chanID   string
		clientID string
		err      error
	}{
		{"disconnect connected client", token, channel.ID, client.ID, nil},
		{"disconnect disconnected client", token, channel.ID, client.ID, clients.ErrNotFound},
		{"disconnect client with wrong credentials", wrong, channel.ID, client.ID, clients.ErrUnauthorizedAccess},
		{"disconnect client from non-existing channel", token, wrong, client.ID, clients.ErrNotFound},
		{"disconnect non-existing client", token, channel.ID, wrong, clients.ErrNotFound},
	}

	for _, tc := range cases {
		err := svc.Disconnect(tc.key, tc.chanID, tc.clientID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccess(t *testing.T) {
	svc := newService(map[string]string{token: email})

	clientID, _ := svc.AddClient(token, client)
	client.ID = clientID
	client.Key = clientID

	channel.Clients = []clients.Client{client}
	chanID, _ := svc.CreateChannel(token, channel)
	channel.ID = chanID

	cases := map[string]struct {
		key     string
		channel string
		err     error
	}{
		"allowed access":              {client.Key, channel.ID, nil},
		"not-connected cannot access": {"", channel.ID, clients.ErrUnauthorizedAccess},
		"access non-existing channel": {client.Key, wrong, clients.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccess(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
