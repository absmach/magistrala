// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	uuidProvider "github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
)

func TestChannelsSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-save@example.com"

	var chid string
	chs := []things.Channel{}
	for i := 1; i <= 5; i++ {
		chid, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chid,
			Owner: email,
		}
		chs = append(chs, ch)
	}

	cases := []struct {
		desc     string
		channels []things.Channel
		err      error
	}{
		{
			desc:     "create new channels",
			channels: chs,
			err:      nil,
		},
		{
			desc:     "create channels that already exist",
			channels: chs,
			err:      things.ErrConflict,
		},
		{
			desc: "create channel with invalid ID",
			channels: []things.Channel{
				things.Channel{
					ID:    "invalid",
					Owner: email,
				},
			},
			err: things.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				things.Channel{
					ID:    chid,
					Owner: email,
					Name:  invalidName,
				},
			},
			err: things.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				things.Channel{
					ID:    chid,
					Owner: email,
					Name:  invalidName,
				},
			},
			err: things.ErrMalformedEntity,
		},
	}

	for _, cc := range cases {
		_, err := channelRepo.Save(context.Background(), cc.channels...)
		assert.Equal(t, cc.err, err, fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	cid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    cid,
		Owner: email,
	}

	schs, _ := chanRepo.Save(context.Background(), ch)
	ch.ID = schs[0].ID

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel things.Channel
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: ch,
			err:     nil,
		},
		{
			desc: "update non-existing channel with existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: email,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update existing channel ID with non-existing user",
			channel: things.Channel{
				ID:    ch.ID,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update non-existing channel with non-existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Update(context.Background(), tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	sths, _ := thingRepo.Save(context.Background(), th)
	th.ID = sths[0].ID

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    chid,
		Owner: email,
	}

	schs, _ := chanRepo.Save(context.Background(), ch)
	ch.ID = schs[0].ID
	chanRepo.Connect(context.Background(), email, []string{ch.ID}, []string{th.ID})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"retrieve channel with existing user": {
			owner: ch.Owner,
			ID:    ch.ID,
			err:   nil,
		},
		"retrieve channel with existing user, non-existing channel": {
			owner: ch.Owner,
			ID:    nonexistentChanID,
			err:   things.ErrNotFound,
		},
		"retrieve channel with non-existing owner": {
			owner: wrongValue,
			ID:    ch.ID,
			err:   things.ErrNotFound,
		},
		"retrieve channel with malformed ID": {
			owner: ch.Owner,
			ID:    wrongValue,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.RetrieveByID(context.Background(), tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-multi-retrieval@example.com"
	name := "channel_name"
	metadata := things.Metadata{
		"field": "value",
	}
	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	offset := uint64(1)
	chNameNum := uint64(3)
	chMetaNum := uint64(3)
	chNameMetaNum := uint64(2)

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		chid, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chid,
			Owner: email,
		}

		// Create Channels with name.
		if i < chNameNum {
			ch.Name = name
		}
		// Create Channels with metadata.
		if i >= chNameNum && i < chNameNum+chMetaNum {
			ch.Metadata = metadata
		}
		// Create Channels with name and metadata.
		if i >= n-chNameMetaNum {
			ch.Metadata = metadata
			ch.Name = name
		}

		chanRepo.Save(context.Background(), ch)
	}

	cases := map[string]struct {
		owner    string
		offset   uint64
		limit    uint64
		name     string
		size     uint64
		total    uint64
		metadata things.Metadata
	}{
		"retrieve all channels with existing owner": {
			owner:  email,
			offset: 0,
			limit:  n,
			size:   n,
			total:  n,
		},
		"retrieve subset of channels with existing owner": {
			owner:  email,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			total:  n,
		},
		"retrieve channels with non-existing owner": {
			owner:  wrongValue,
			offset: n / 2,
			limit:  n,
			size:   0,
			total:  0,
		},
		"retrieve channels with existing name": {
			owner:  email,
			offset: offset,
			limit:  n,
			name:   name,
			size:   chNameNum + chNameMetaNum - offset,
			total:  chNameNum + chNameMetaNum,
		},
		"retrieve all channels with non-existing name": {
			owner:  email,
			offset: 0,
			limit:  n,
			name:   "wrong",
			size:   0,
			total:  0,
		},
		"retrieve all channels with existing metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			size:     chMetaNum + chNameMetaNum,
			total:    chMetaNum + chNameMetaNum,
			metadata: metadata,
		},
		"retrieve all channels with non-existing metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			total:    0,
			metadata: wrongMeta,
		},
		"retrieve all channels with existing name and metadata": {
			owner:    email,
			offset:   0,
			limit:    n,
			size:     chNameMetaNum,
			total:    chNameMetaNum,
			name:     name,
			metadata: metadata,
		},
	}

	for desc, tc := range cases {
		page, err := chanRepo.RetrieveAll(context.Background(), tc.owner, tc.offset, tc.limit, tc.name, tc.metadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveByThing(t *testing.T) {
	email := "channel-multi-retrieval-by-thing@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sths, err := thingRepo.Save(context.Background(), things.Thing{
		ID:    thid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	tid := sths[0].ID

	n := uint64(10)
	chsDisconNum := uint64(1)

	for i := uint64(0); i < n; i++ {
		chid, err := uuidProvider.New().ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		ch := things.Channel{
			ID:    chid,
			Owner: email,
		}
		schs, err := chanRepo.Save(context.Background(), ch)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		cid := schs[0].ID

		// Don't connect last Channel
		if i == n-chsDisconNum {
			break
		}

		err = chanRepo.Connect(context.Background(), email, []string{cid}, []string{tid})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner     string
		thing     string
		offset    uint64
		limit     uint64
		connected bool
		size      uint64
		err       error
	}{
		"retrieve all channels by thing with existing owner": {
			owner:     email,
			thing:     tid,
			offset:    0,
			limit:     n,
			connected: true,
			size:      n - chsDisconNum,
		},
		"retrieve subset of channels by thing with existing owner": {
			owner:     email,
			thing:     tid,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      (n / 2) - chsDisconNum,
		},
		"retrieve channels by thing with non-existing owner": {
			owner:     wrongValue,
			thing:     tid,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      0,
		},
		"retrieve channels by non-existent thing": {
			owner:     email,
			thing:     nonexistentThingID,
			offset:    0,
			limit:     n,
			connected: true,
			size:      0,
		},
		"retrieve channels with malformed UUID": {
			owner:     email,
			thing:     wrongValue,
			offset:    0,
			limit:     n,
			connected: true,
			size:      0,
			err:       things.ErrNotFound,
		},
		"retrieve all non connected channels by thing with existing owner": {
			owner:     email,
			thing:     tid,
			offset:    0,
			limit:     n,
			connected: false,
			size:      chsDisconNum,
		},
	}

	for desc, tc := range cases {
		page, err := chanRepo.RetrieveByThing(context.Background(), tc.owner, tc.thing, tc.offset, tc.limit, tc.connected)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	schs, _ := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	chanID := schs[0].ID

	// show that the removal works the same for both existing and non-existing
	// (removed) channel
	for i := 0; i < 2; i++ {
		err := chanRepo.Remove(context.Background(), email, chanID)
		require.Nil(t, err, fmt.Sprintf("#%d: failed to remove channel due to: %s", i, err))

		_, err = chanRepo.RetrieveByID(context.Background(), email, chanID)
		require.Equal(t, things.ErrNotFound, err, fmt.Sprintf("#%d: expected %s got %s", i, things.ErrNotFound, err))
	}
}

func TestConnect(t *testing.T) {
	email := "channel-connect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:       thid,
		Owner:    email,
		Key:      thkey,
		Metadata: things.Metadata{},
	}
	sths, _ := thingRepo.Save(context.Background(), thing)
	thingID := sths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	schs, _ := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	chanID := schs[0].ID

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		owner   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "connect existing user, channel and thing",
			owner:   email,
			chanID:  chanID,
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "connect connected channel and thing",
			owner:   email,
			chanID:  chanID,
			thingID: thingID,
			err:     things.ErrConflict,
		},
		{
			desc:    "connect with non-existing user",
			owner:   wrongValue,
			chanID:  chanID,
			thingID: thingID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "connect non-existing channel",
			owner:   email,
			chanID:  nonexistentChanID,
			thingID: thingID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "connect non-existing thing",
			owner:   email,
			chanID:  chanID,
			thingID: nonexistentThingID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Connect(context.Background(), tc.owner, []string{tc.chanID}, []string{tc.thingID})
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:       thid,
		Owner:    email,
		Key:      thkey,
		Metadata: map[string]interface{}{},
	}
	sths, _ := thingRepo.Save(context.Background(), thing)
	thingID := sths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	schs, _ := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	chanID := schs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chanID}, []string{thingID})

	nonexistentThingID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		owner   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "disconnect connected thing",
			owner:   email,
			chanID:  chanID,
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "disconnect non-connected thing",
			owner:   email,
			chanID:  chanID,
			thingID: thingID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing user",
			owner:   wrongValue,
			chanID:  chanID,
			thingID: thingID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing channel",
			owner:   email,
			chanID:  nonexistentChanID,
			thingID: thingID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing thing",
			owner:   email,
			chanID:  chanID,
			thingID: nonexistentThingID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(context.Background(), tc.owner, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestHasThing(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	sths, _ := thingRepo.Save(context.Background(), thing)
	thingID := sths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	schs, _ := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	chanID := schs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chanID}, []string{thingID})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		chanID    string
		key       string
		hasAccess bool
	}{
		"access check for thing that has access": {
			chanID:    chanID,
			key:       thing.Key,
			hasAccess: true,
		},
		"access check for thing without access": {
			chanID:    chanID,
			key:       wrongValue,
			hasAccess: false,
		},
		"access check for non-existing channel": {
			chanID:    nonexistentChanID,
			key:       thing.Key,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.HasThing(context.Background(), tc.chanID, tc.key)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}

func TestHasThingByID(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}
	sths, _ := thingRepo.Save(context.Background(), thing)
	thingID := sths[0].ID

	disconnectedThID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	disconnectedThKey, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	disconnectedThing := things.Thing{
		ID:    disconnectedThID,
		Owner: email,
		Key:   disconnectedThKey,
	}
	sths, _ = thingRepo.Save(context.Background(), disconnectedThing)
	disconnectedThingID := sths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chid, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	schs, _ := chanRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	chanID := schs[0].ID
	chanRepo.Connect(context.Background(), email, []string{chanID}, []string{thingID})

	nonexistentChanID, err := uuidProvider.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		chanID    string
		thingID   string
		hasAccess bool
	}{
		"access check for thing that has access": {
			chanID:    chanID,
			thingID:   thingID,
			hasAccess: true,
		},
		"access check for thing without access": {
			chanID:    chanID,
			thingID:   disconnectedThingID,
			hasAccess: false,
		},
		"access check for non-existing channel": {
			chanID:    nonexistentChanID,
			thingID:   thingID,
			hasAccess: false,
		},
		"access check for non-existing thing": {
			chanID:    chanID,
			thingID:   wrongValue,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		err := chanRepo.HasThingByID(context.Background(), tc.chanID, tc.thingID)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}
