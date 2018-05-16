package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/mainflux/mainflux/things/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChannelSave(t *testing.T) {
	email := "channel-save@example.com"
	idp := uuid.New()
	channelRepo := postgres.NewChannelRepository(db, testLog)

	channel := things.Channel{ID: idp.ID(), Owner: email}
	_, err := channelRepo.Save(channel)
	hasErr := err != nil

	assert.False(t, hasErr, fmt.Sprintf("create new channel: expected false got %t", hasErr))
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"
	idp := uuid.New()
	chanRepo := postgres.NewChannelRepository(db, testLog)

	c := things.Channel{ID: idp.ID(), Owner: email}
	chanRepo.Save(c)

	cases := map[string]struct {
		channel things.Channel
		err     error
	}{
		"existing channel":                            {c, nil},
		"non-existing channel with existing user":     {things.Channel{ID: wrong, Owner: email}, things.ErrNotFound},
		"non-existing channel with non-existing user": {things.Channel{ID: wrong, Owner: wrong}, things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := chanRepo.Update(tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"
	idp := uuid.New()
	chanRepo := postgres.NewChannelRepository(db, testLog)

	c := things.Channel{ID: idp.ID(), Owner: email}
	chanRepo.Save(c)

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"existing user":                       {c.Owner, c.ID, nil},
		"existing user, non-existing channel": {c.Owner, wrong, things.ErrNotFound},
		"non-existing owner":                  {wrong, c.ID, things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := chanRepo.One(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	email := "channel-multi-retrieval@example.com"
	idp := uuid.New()
	chanRepo := postgres.NewChannelRepository(db, testLog)

	n := 10

	for i := 0; i < n; i++ {
		c := things.Channel{ID: idp.ID(), Owner: email}
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
	idp := uuid.New()
	chanRepo := postgres.NewChannelRepository(db, testLog)
	chanID, _ := chanRepo.Save(things.Channel{ID: idp.ID(), Owner: email})

	// show that the removal works the same for both existing and non-existing
	// (removed) channel
	for i := 0; i < 2; i++ {
		if err := chanRepo.Remove(email, chanID); err != nil {
			t.Fatalf("#%d: failed to remove channel due to: %s", i, err)
		}

		if _, err := chanRepo.One(email, chanID); err != things.ErrNotFound {
			t.Fatalf("#%d: expected %s got %s", i, things.ErrNotFound, err)
		}
	}
}

func TestConnect(t *testing.T) {
	email := "channel-connect@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}
	thingRepo.Save(thing)

	chanRepo := postgres.NewChannelRepository(db, testLog)
	chanID, _ := chanRepo.Save(things.Channel{ID: idp.ID(), Owner: email})

	cases := []struct {
		desc    string
		owner   string
		chanID  string
		thingID string
		err     error
	}{
		{"existing user, channel and thing", email, chanID, thing.ID, nil},
		{"connected channel and thing", email, chanID, thing.ID, nil},
		{"with non-existing user", wrong, chanID, thing.ID, things.ErrNotFound},
		{"non-existing channel", email, wrong, thing.ID, things.ErrNotFound},
		{"non-existing thing", email, chanID, wrong, things.ErrNotFound},
	}

	for _, tc := range cases {
		err := chanRepo.Connect(tc.owner, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)
	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}
	thingRepo.Save(thing)

	chanRepo := postgres.NewChannelRepository(db, testLog)
	chanID, _ := chanRepo.Save(things.Channel{ID: idp.ID(), Owner: email})
	chanRepo.Connect(email, chanID, thing.ID)

	cases := []struct {
		desc    string
		owner   string
		chanID  string
		thingID string
		err     error
	}{
		{"connected thing", email, chanID, thing.ID, nil},
		{"non-connected thing", email, chanID, thing.ID, things.ErrNotFound},
		{"non-existing user", wrong, chanID, thing.ID, things.ErrNotFound},
		{"non-existing channel", email, wrong, thing.ID, things.ErrNotFound},
		{"non-existing thing", email, chanID, wrong, things.ErrNotFound},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(tc.owner, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestHasThing(t *testing.T) {
	email := "channel-access-check@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)
	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}
	thingRepo.Save(thing)

	chanRepo := postgres.NewChannelRepository(db, testLog)
	chanID, _ := chanRepo.Save(things.Channel{ID: idp.ID(), Owner: email})
	chanRepo.Connect(email, chanID, thing.ID)

	cases := map[string]struct {
		chanID    string
		key       string
		hasAccess bool
	}{
		"thing that has access":                {chanID, thing.Key, true},
		"thing without access":                 {chanID, wrong, false},
		"check access to non-existing channel": {wrong, thing.Key, false},
	}

	for desc, tc := range cases {
		_, err := chanRepo.HasThing(tc.chanID, tc.key)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}
