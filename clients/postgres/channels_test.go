package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/clients"
	"github.com/mainflux/mainflux/clients/postgres"
	"github.com/stretchr/testify/assert"
)

func TestChannelSave(t *testing.T) {
	email := "channel-save@example.com"
	channel := clients.Channel{Owner: email}

	channelRepo := postgres.NewChannelRepository(db)

	_, err := channelRepo.Save(channel)
	hasErr := err != nil
	assert.False(t, hasErr, fmt.Sprintf("create new channel: expected false got %t", hasErr))
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"

	chanRepo := postgres.NewChannelRepository(db)

	c := clients.Channel{Owner: email}
	id, _ := chanRepo.Save(c)
	c.ID = id

	cases := map[string]struct {
		channel clients.Channel
		err     error
	}{
		"existing channel":                            {c, nil},
		"non-existing channel with existing user":     {clients.Channel{ID: wrong, Owner: email}, clients.ErrNotFound},
		"non-existing channel with non-existing user": {clients.Channel{ID: wrong, Owner: wrong}, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := chanRepo.Update(tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"

	chanRepo := postgres.NewChannelRepository(db)

	c := clients.Channel{Owner: email}
	id, _ := chanRepo.Save(c)

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"existing user":                       {c.Owner, id, nil},
		"existing user, non-existing channel": {c.Owner, wrong, clients.ErrNotFound},
		"non-existing owner":                  {wrong, id, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := chanRepo.One(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	email := "channel-multi-retrieval@example.com"

	chanRepo := postgres.NewChannelRepository(db)

	n := 10

	for i := 0; i < n; i++ {
		c := clients.Channel{Owner: email}
		chanRepo.Save(c)
	}

	cases := map[string]struct {
		owner  string
		offset int
		limit  int
		size   int
	}{
		"existing owner":     {email, 0, n, n},
		"non-existing owner": {wrong, 1, 6, 0},
	}

	for desc, tc := range cases {
		size := len(chanRepo.All(tc.owner, tc.offset, tc.limit))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"

	chanRepo := postgres.NewChannelRepository(db)
	chanID, _ := chanRepo.Save(clients.Channel{Owner: email})

	// show that the removal works the same for both existing and non-existing
	// (removed) channel
	for i := 0; i < 2; i++ {
		if err := chanRepo.Remove(email, chanID); err != nil {
			t.Fatalf("#%d: failed to remove channel due to: %s", i, err)
		}

		if _, err := chanRepo.One(email, chanID); err != clients.ErrNotFound {
			t.Fatalf("#%d: expected %s got %s", i, clients.ErrNotFound, err)
		}
	}
}

func TestChannelConnect(t *testing.T) {
	email := "channel-connect@example.com"

	clientRepo := postgres.NewClientRepository(db)
	client := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanID, _ := chanRepo.Save(clients.Channel{Owner: email})

	cases := map[string]struct {
		owner    string
		chanID   string
		clientID string
		err      error
	}{
		"existing user, channel and client": {email, chanID, client.ID, nil},
		"with non-existing user":            {wrong, chanID, client.ID, clients.ErrNotFound},
		"non-existing channel":              {email, wrong, client.ID, clients.ErrNotFound},
		"non-existing client":               {email, chanID, wrong, clients.ErrNotFound},
	}

	for desc, tc := range cases {
		err := chanRepo.Connect(tc.owner, tc.chanID, tc.clientID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChannelDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"

	clientRepo := postgres.NewClientRepository(db)
	client := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanID, _ := chanRepo.Save(clients.Channel{Owner: email})

	chanRepo.Connect(email, chanID, client.ID)

	cases := []struct {
		desc     string
		owner    string
		chanID   string
		clientID string
		err      error
	}{
		{"connected client", email, chanID, client.ID, nil},
		{"non-connected client", email, chanID, client.ID, clients.ErrNotFound},
		{"non-existing user", wrong, chanID, client.ID, clients.ErrNotFound},
		{"non-existing channel", email, wrong, client.ID, clients.ErrNotFound},
		{"non-existing client", email, chanID, wrong, clients.ErrNotFound},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(tc.owner, tc.chanID, tc.clientID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChannelAccessCheck(t *testing.T) {
	email := "channel-access-check@example.com"

	clientRepo := postgres.NewClientRepository(db)
	client := clients.Client{
		ID:    clientRepo.ID(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanID, _ := chanRepo.Save(clients.Channel{Owner: email})

	chanRepo.Connect(email, chanID, client.ID)

	cases := map[string]struct {
		chanID    string
		clientID  string
		hasAccess bool
	}{
		"client that has access":               {chanID, client.ID, true},
		"client without access":                {chanID, wrong, false},
		"check access to non-existing channel": {wrong, client.ID, false},
	}

	for desc, tc := range cases {
		hasAccess := chanRepo.HasClient(tc.chanID, tc.clientID)
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}
