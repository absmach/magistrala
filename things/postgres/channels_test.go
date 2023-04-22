// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
)

func TestChannelsSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-save@example.com"

	chs := []things.Channel{}
	for i := 1; i <= 5; i++ {
		id, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    id,
			Owner: email,
		}
		chs = append(chs, ch)
	}
	id := chs[0].ID

	cases := []struct {
		desc     string
		channels []things.Channel
		response []things.Channel
		err      error
	}{
		{
			desc:     "create new channels",
			channels: chs,
			response: chs,
			err:      nil,
		},
		{
			desc:     "create channels that already exist",
			channels: chs,
			response: []things.Channel{},
			err:      errors.ErrConflict,
		},
		{
			desc: "create channel with invalid ID",
			channels: []things.Channel{
				{ID: "invalid", Owner: email},
			},
			response: []things.Channel{},
			err:      errors.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			response: []things.Channel{},
			err:      errors.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			response: []things.Channel{},
			err:      errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := channelRepo.Save(context.Background(), tc.channels...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	id, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    id,
		Owner: email,
	}

	chs, err := chanRepo.Save(context.Background(), ch)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch.ID = chs[0].ID

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

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
			err: errors.ErrNotFound,
		},
		{
			desc: "update existing channel ID with non-existing user",
			channel: things.Channel{
				ID:    ch.ID,
				Owner: wrongValue,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update non-existing channel with non-existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: wrongValue,
			},
			err: errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Update(context.Background(), tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:    thID,
		Owner: email,
		Key:   thkey,
	}
	ths, _ := thingRepo.Save(context.Background(), th)
	th.ID = ths[0].ID

	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:       chID,
		Owner:    email,
		Metadata: make(map[string]interface{}),
	}
	chs, _ := chanRepo.Save(context.Background(), ch)
	ch.ID = chs[0].ID

	err = chanRepo.Connect(context.Background(), email, []string{ch.ID}, []string{th.ID})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error while connecting to service: %s", err))

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		owner    string
		ID       string
		response things.Channel
		err      error
	}{
		{
			desc:     "retrieve channel with existing user",
			owner:    ch.Owner,
			ID:       ch.ID,
			response: ch,
			err:      nil,
		},
		{
			desc:     "retrieve channel with existing user, non-existing channel",
			owner:    ch.Owner,
			ID:       nonexistentChanID,
			response: things.Channel{},
			err:      errors.ErrNotFound,
		},
		{
			desc:     "retrieve channel with malformed ID",
			owner:    ch.Owner,
			ID:       wrongValue,
			response: things.Channel{},
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		resp, err := chanRepo.RetrieveByID(context.Background(), tc.owner, tc.ID)
		assert.Equal(t, tc.response, resp, fmt.Sprintf("%s: got incorrect channel from RetrieveByID()", tc.desc))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
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
	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		chID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chID,
			Owner: email,
		}

		// Create Channels with name.
		if i < nameNum {
			ch.Name = fmt.Sprintf("%s-%d", name, i)
		}
		// Create Channels with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			ch.Metadata = metadata
		}
		// Create Channels with name and metadata.
		if i >= n-nameMetaNum {
			ch.Metadata = metadata
			ch.Name = name
		}

		_, err = chanRepo.Save(context.Background(), ch)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error while saving channels: %s", err))
	}

	cases := []struct {
		desc         string
		owner        string
		size         uint64
		pageMetadata things.PageMetadata
	}{
		{
			desc:  "retrieve all channels with existing owner",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
		},
		{
			desc:  "retrieve subset of channels with existing owner",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
				Total:  n,
			},
			size: n / 2,
		},
		{
			desc:  "retrieve channels with non-existing owner",
			owner: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
				Total:  0,
			},
			size: 0,
		},
		{
			desc:  "retrieve channels with existing name",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
				Name:   name,
				Total:  nameNum + nameMetaNum,
			},
			size: nameNum + nameMetaNum - offset,
		},
		{
			desc:  "retrieve all channels with non-existing name",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Total:  0,
			},
			size: 0,
		},
		{
			desc:  "retrieve all channels with existing metadata",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
				Total:    metaNum + nameMetaNum,
			},
			size: metaNum + nameMetaNum,
		},
		{
			desc:  "retrieve all channels with non-existing metadata",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
				Total:    0,
			},
		},
		{
			desc:  "retrieve all channels with existing name and metadata",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Name:     name,
				Metadata: metadata,
				Total:    nameMetaNum,
			},
			size: nameMetaNum,
		},
		{
			desc:  "retrieve channels sorted by name ascendent",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
		},
		{
			desc:  "retrieve channels sorted by name descendent",
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
		},
	}

	for _, tc := range cases {
		page, err := chanRepo.RetrieveAll(context.Background(), tc.owner, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", tc.desc, tc.pageMetadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", tc.desc, err))

		// Check if Channels list have been sorted properly
		testSortChannels(t, tc.pageMetadata, page.Channels)
	}
}

func TestRetrieveByThing(t *testing.T) {
	email := "channel-multi-retrieval-by-thing@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ths, err := thingRepo.Save(context.Background(), things.Thing{
		ID:    thID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID = ths[0].ID

	n := uint64(10)
	chsDisconNum := uint64(1)

	for i := uint64(0); i < n; i++ {
		chID, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		ch := things.Channel{
			ID:    chID,
			Owner: email,
		}
		schs, err := chanRepo.Save(context.Background(), ch)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		cid := schs[0].ID

		// Don't connect last Channel
		if i == n-chsDisconNum {
			break
		}

		err = chanRepo.Connect(context.Background(), email, []string{cid}, []string{thID})
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	nonexistentThingID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		owner        string
		thID         string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "retrieve all channels by thing with existing owner",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n - chsDisconNum,
		},
		{
			desc:  "retrieve subset of channels by thing with existing owner",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: (n / 2) - chsDisconNum,
		},
		{
			desc:  "retrieve channels by thing with non-existing owner",
			owner: wrongValue,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: 0,
		},
		{
			desc:  "retrieve channels by non-existent thing",
			owner: email,
			thID:  nonexistentThingID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
		},
		{
			desc:  "retrieve channels with malformed UUID",
			owner: email,
			thID:  wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrNotFound,
		},
		{
			desc:  "retrieve all non connected channels by thing with existing owner",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
			},
			size: chsDisconNum,
		},
		{
			desc:  "retrieve all channels by thing sorted by name ascendent",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n - chsDisconNum,
		},
		{
			desc:  "retrieve all non-connected channels by thing sorted by name ascendent",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "asc",
			},
			size: chsDisconNum,
		},
		{
			desc:  "retrieve all channels by thing sorted by name descendent",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n - chsDisconNum,
		},
		{
			desc:  "retrieve all non-connected channels by thing sorted by name descendent",
			owner: email,
			thID:  thID,
			pageMetadata: things.PageMetadata{
				Offset:       0,
				Limit:        n,
				Disconnected: true,
				Order:        "name",
				Dir:          "desc",
			},
			size: chsDisconNum,
		},
	}

	for _, tc := range cases {
		page, err := chanRepo.RetrieveByThing(context.Background(), tc.owner, tc.thID, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", tc.desc, err))

		// Check if Channels by Thing list have been sorted properly
		testSortChannels(t, tc.pageMetadata, page.Channels)
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	// show that the removal works the same for both existing and non-existing (removed) channel
	for i := 0; i < 2; i++ {
		err := chanRepo.Remove(context.Background(), email, chID)
		assert.Nil(t, err, fmt.Sprintf("#%d: failed to remove channel due to: %s", i, err))

		resp, err := chanRepo.RetrieveByID(context.Background(), email, chID)
		assert.Equal(t, things.Channel{}, resp)
		assert.True(t, errors.Contains(err, errors.ErrNotFound), fmt.Sprintf("#%d: expected %s got %s", i, errors.ErrNotFound, err))
	}
}

func TestConnect(t *testing.T) {
	email := "channel-connect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:       thID,
		Owner:    email,
		Key:      thkey,
		Metadata: things.Metadata{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	nonexistentThingID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chID  string
		thID  string
		err   error
	}{
		{
			desc:  "connect existing user, channel and thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   nil,
		},
		{
			desc:  "connect connected channel and thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrConflict,
		},
		{
			desc:  "connect with non-existing user",
			owner: wrongValue,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "connect non-existing channel",
			owner: email,
			chID:  nonexistentChanID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "connect non-existing thing",
			owner: email,
			chID:  chID,
			thID:  nonexistentThingID,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Connect(context.Background(), tc.owner, []string{tc.chID}, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:       thID,
		Owner:    email,
		Key:      thkey,
		Metadata: map[string]interface{}{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	err = chanRepo.Connect(context.Background(), email, []string{chID}, []string{thID})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error while connecting to service: %s", err))

	nonexistentThingID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chID  string
		thID  string
		err   error
	}{
		{
			desc:  "disconnect connected thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   nil,
		},
		{
			desc:  "disconnect non-connected thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing user",
			owner: wrongValue,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing channel",
			owner: email,
			chID:  nonexistentChanID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing thing",
			owner: email,
			chID:  chID,
			thID:  nonexistentThingID,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(context.Background(), tc.owner, []string{tc.chID}, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestHasThing(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:    thID,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	err = chanRepo.Connect(context.Background(), email, []string{chID}, []string{thID})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error while connecting to service: %s", err))

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		chID      string
		key       string
		hasAccess bool
	}{
		{
			desc:      "access check for thing that has access",
			chID:      chID,
			key:       th.Key,
			hasAccess: true,
		},
		{
			desc:      "access check for thing without access",
			chID:      chID,
			key:       wrongValue,
			hasAccess: false,
		},
		{
			desc:      "access check for non-existing channel",
			chID:      nonexistentChanID,
			key:       th.Key,
			hasAccess: false,
		},
	}

	for _, tc := range cases {
		_, err := chanRepo.HasThing(context.Background(), tc.chID, tc.key)
		assert.Equal(t, tc.hasAccess, err == nil, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.hasAccess, err == nil))
	}
}

func TestHasThingByID(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:    thID,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	disconnectedThID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	disconnectedThKey, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	disconnectedThing := things.Thing{
		ID:    disconnectedThID,
		Owner: email,
		Key:   disconnectedThKey,
	}
	ths, err = thingRepo.Save(context.Background(), disconnectedThing)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	disconnectedThingID := ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	err = chanRepo.Connect(context.Background(), email, []string{chID}, []string{thID})
	assert.Nil(t, err, fmt.Sprintf("got unexpected error while connecting to service: %s", err))

	nonexistentChanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		chID      string
		thID      string
		hasAccess bool
	}{
		{
			desc:      "access check for thing that has access",
			chID:      chID,
			thID:      thID,
			hasAccess: true,
		},
		{
			desc:      "access check for thing without access",
			chID:      chID,
			thID:      disconnectedThingID,
			hasAccess: false,
		},
		{
			desc:      "access check for non-existing channel",
			chID:      nonexistentChanID,
			thID:      thID,
			hasAccess: false,
		},
		{
			desc:      "access check for non-existing thing",
			chID:      chID,
			thID:      wrongValue,
			hasAccess: false,
		},
	}

	for _, tc := range cases {
		err := chanRepo.HasThingByID(context.Background(), tc.chID, tc.thID)
		assert.Equal(t, tc.hasAccess, err == nil, fmt.Sprintf("%s: expected %t got %t\n", tc.desc, tc.hasAccess, err == nil))
	}
}

func testSortChannels(t *testing.T, pm things.PageMetadata, chs []things.Channel) {
	switch pm.Order {
	case "name":
		current := chs[0]
		for _, res := range chs {
			if pm.Dir == "asc" {
				assert.GreaterOrEqual(t, res.Name, current.Name)
			}
			if pm.Dir == "desc" {
				assert.GreaterOrEqual(t, current.Name, res.Name)
			}
			current = res
		}
	default:
		break
	}
}
