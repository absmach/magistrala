package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/manager"
	"github.com/mainflux/mainflux/manager/postgres"
	"github.com/stretchr/testify/assert"
)

func TestChannelSave(t *testing.T) {
	email := "channel-save@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	c1 := manager.Channel{Owner: email}
	c2 := manager.Channel{Owner: wrong}

	cases := map[string]struct {
		channel manager.Channel
		hasErr  bool
	}{
		"new channel, existing user":     {c1, false},
		"new channel, non-existing user": {c2, true},
	}

	channelRepo := postgres.NewChannelRepository(db)

	for desc, tc := range cases {
		_, err := channelRepo.Save(tc.channel)
		hasErr := err != nil
		assert.Equal(t, tc.hasErr, hasErr, fmt.Sprintf("%s: expected %t got %t", desc, tc.hasErr, hasErr))
	}
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	chanRepo := postgres.NewChannelRepository(db)

	c := manager.Channel{Owner: email}
	id, _ := chanRepo.Save(c)
	c.ID = id

	cases := map[string]struct {
		channel manager.Channel
		err     error
	}{
		"existing channel":                            {c, nil},
		"non-existing channel with existing user":     {manager.Channel{ID: wrong, Owner: email}, manager.ErrNotFound},
		"non-existing channel with non-existing user": {manager.Channel{ID: wrong, Owner: wrong}, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := chanRepo.Update(tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	chanRepo := postgres.NewChannelRepository(db)

	c := manager.Channel{Owner: email}
	id, _ := chanRepo.Save(c)

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"existing user":                       {c.Owner, id, nil},
		"existing user, non-existing channel": {c.Owner, wrong, manager.ErrNotFound},
		"non-existing owner":                  {wrong, id, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := chanRepo.One(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	email := "channel-multi-retrieval@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	chanRepo := postgres.NewChannelRepository(db)

	n := 10

	for i := 0; i < n; i++ {
		c := manager.Channel{Owner: email}
		chanRepo.Save(c)
	}

	cases := map[string]struct {
		owner string
		len   int
	}{
		"existing owner":     {email, n},
		"non-existing owner": {wrong, 0},
	}

	for desc, tc := range cases {
		n := len(chanRepo.All(tc.owner))
		assert.Equal(t, tc.len, n, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.len, n))
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	chanRepo := postgres.NewChannelRepository(db)
	chanId, _ := chanRepo.Save(manager.Channel{Owner: email})

	// show that the removal works the same for both existing and non-existing
	// (removed) channel
	for i := 0; i < 2; i++ {
		if err := chanRepo.Remove(email, chanId); err != nil {
			t.Fatalf("#%d: failed to remove channel due to: %s", i, err)
		}

		if _, err := chanRepo.One(email, chanId); err != manager.ErrNotFound {
			t.Fatalf("#%d: expected %s got %s", i, manager.ErrNotFound, err)
		}
	}
}

func TestChannelConnect(t *testing.T) {
	email := "channel-connect@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	clientRepo := postgres.NewClientRepository(db)
	client := manager.Client{
		ID:    clientRepo.Id(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanId, _ := chanRepo.Save(manager.Channel{Owner: email})

	cases := map[string]struct {
		owner    string
		chanId   string
		clientId string
		err      error
	}{
		"existing user, channel and client": {email, chanId, client.ID, nil},
		"with non-existing user":            {wrong, chanId, client.ID, manager.ErrNotFound},
		"non-existing channel":              {email, wrong, client.ID, manager.ErrNotFound},
		"non-existing client":               {email, chanId, wrong, manager.ErrNotFound},
	}

	for desc, tc := range cases {
		err := chanRepo.Connect(tc.owner, tc.chanId, tc.clientId)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestChannelDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	clientRepo := postgres.NewClientRepository(db)
	client := manager.Client{
		ID:    clientRepo.Id(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanId, _ := chanRepo.Save(manager.Channel{Owner: email})

	chanRepo.Connect(email, chanId, client.ID)

	cases := []struct {
		desc     string
		owner    string
		chanId   string
		clientId string
		err      error
	}{
		{"connected client", email, chanId, client.ID, nil},
		{"non-connected client", email, chanId, client.ID, manager.ErrNotFound},
		{"non-existing user", wrong, chanId, client.ID, manager.ErrNotFound},
		{"non-existing channel", email, wrong, client.ID, manager.ErrNotFound},
		{"non-existing client", email, chanId, wrong, manager.ErrNotFound},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(tc.owner, tc.chanId, tc.clientId)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChannelAccessCheck(t *testing.T) {
	email := "channel-access-check@example.com"

	userRepo := postgres.NewUserRepository(db)
	userRepo.Save(manager.User{email, "pass"})

	clientRepo := postgres.NewClientRepository(db)
	client := manager.Client{
		ID:    clientRepo.Id(),
		Owner: email,
	}
	clientRepo.Save(client)

	chanRepo := postgres.NewChannelRepository(db)
	chanId, _ := chanRepo.Save(manager.Channel{Owner: email})

	chanRepo.Connect(email, chanId, client.ID)

	cases := map[string]struct {
		chanId    string
		clientId  string
		hasAccess bool
	}{
		"client that has access":               {chanId, client.ID, true},
		"client without access":                {chanId, wrong, false},
		"check access to non-existing channel": {wrong, client.ID, false},
	}

	for desc, tc := range cases {
		hasAccess := chanRepo.HasClient(tc.chanId, tc.clientId)
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}
