//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/mainflux/mainflux/things/uuid"
	"github.com/stretchr/testify/assert"
)

const maxNameSize = 1024

var invalidName = strings.Repeat("m", maxNameSize+1)

func TestThingSave(t *testing.T) {
	thingRepo := postgres.NewThingRepository(db)

	email := "thing-save@example.com"

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentThingKey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	cases := []struct {
		desc  string
		thing things.Thing
		err   error
	}{
		{
			desc:  "create new thing",
			thing: thing,
			err:   nil,
		},
		{
			desc:  "create thing with conflicting key",
			thing: thing,
			err:   things.ErrConflict,
		},
		{
			desc: "create thing with invalid ID",
			thing: things.Thing{
				ID:    "invalid",
				Owner: email,
				Key:   thkey,
			},
			err: things.ErrMalformedEntity,
		},
		{
			desc: "create thing with invalid Key",
			thing: things.Thing{
				ID:    thid,
				Owner: email,
				Key:   nonexistentThingKey,
			},
			err: things.ErrConflict,
		},
		{
			desc: "create thing with invalid name",
			thing: things.Thing{
				ID:    thid,
				Owner: email,
				Key:   thkey,
				Name:  invalidName,
			},
			err: things.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := thingRepo.Save(context.Background(), tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestThingUpdate(t *testing.T) {
	thingRepo := postgres.NewThingRepository(db)

	email := "thing-update@example.com"
	validName := "mfx_device"

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	id, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = id

	nonexistentThingID, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		thing things.Thing
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: thing,
			err:   nil,
		},
		{
			desc: "update non-existing thing with existing user",
			thing: things.Thing{
				ID:    nonexistentThingID,
				Owner: email,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update existing thing ID with non-existing user",
			thing: things.Thing{
				ID:    id,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update non-existing thing with non-existing user",
			thing: things.Thing{
				ID:    nonexistentThingID,
				Owner: wrongValue,
			},
			err: things.ErrNotFound,
		},
		{
			desc: "update thing with valid name",
			thing: things.Thing{
				ID:    thid,
				Owner: email,
				Key:   thkey,
				Name:  validName,
			},
			err: nil,
		},
		{
			desc: "update thing with invalid name",
			thing: things.Thing{
				ID:    thid,
				Owner: email,
				Key:   thkey,
				Name:  invalidName,
			},
			err: things.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := thingRepo.Update(context.Background(), tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	email := "thing-update=key@example.com"
	newKey := "new-key"
	thingRepo := postgres.NewThingRepository(db)

	ethid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ethkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	existingThing := things.Thing{
		ID:    ethid,
		Owner: email,
		Key:   ethkey,
	}
	existingID, _ := thingRepo.Save(context.Background(), existingThing)
	existingThing.ID = existingID

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	id, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = id

	nonexistentThingID, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		id    string
		key   string
		err   error
	}{
		{
			desc:  "update key of an existing thing",
			owner: thing.Owner,
			id:    thing.ID,
			key:   newKey,
			err:   nil,
		},
		{
			desc:  "update key of a non-existing thing with existing user",
			owner: thing.Owner,
			id:    nonexistentThingID,
			key:   newKey,
			err:   things.ErrNotFound,
		},
		{
			desc:  "update key of an existing thing with non-existing user",
			owner: wrongValue,
			id:    thing.ID,
			key:   newKey,
			err:   things.ErrNotFound,
		},
		{
			desc:  "update key of a non-existing thing with non-existing user",
			owner: wrongValue,
			id:    nonexistentThingID,
			key:   newKey,
			err:   things.ErrNotFound,
		},
		{
			desc:  "update key with existing key value",
			owner: thing.Owner,
			id:    thing.ID,
			key:   existingThing.Key,
			err:   things.ErrConflict,
		},
	}

	for _, tc := range cases {
		err := thingRepo.UpdateKey(context.Background(), tc.owner, tc.id, tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleThingRetrieval(t *testing.T) {
	email := "thing-single-retrieval@example.com"
	thingRepo := postgres.NewThingRepository(db)

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	id, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = id

	nonexistentThingID, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"retrieve thing with existing user": {
			owner: thing.Owner,
			ID:    thing.ID,
			err:   nil,
		},
		"retrieve non-existing thing with existing user": {
			owner: thing.Owner,
			ID:    nonexistentThingID,
			err:   things.ErrNotFound,
		},
		"retrieve thing with non-existing owner": {
			owner: wrongValue,
			ID:    thing.ID,
			err:   things.ErrNotFound,
		},
		"retrieve thing with malformed ID": {
			owner: thing.Owner,
			ID:    wrongValue,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := thingRepo.RetrieveByID(context.Background(), tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRetrieveByKey(t *testing.T) {
	email := "thing-retrieved-by-key@example.com"
	thingRepo := postgres.NewThingRepository(db)

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	id, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = id

	cases := map[string]struct {
		key string
		ID  string
		err error
	}{
		"retrieve existing thing by key": {
			key: thing.Key,
			ID:  thing.ID,
			err: nil,
		},
		"retrieve non-existent thing by key": {
			key: wrongValue,
			ID:  "",
			err: things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := thingRepo.RetrieveByKey(context.Background(), tc.key)
		assert.Equal(t, tc.ID, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.ID, id))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiThingRetrieval(t *testing.T) {
	email := "thing-multi-retrieval@example.com"
	name := "mainflux"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db)

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		thid, err := idp.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thkey, err := idp.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		th := things.Thing{
			Owner: email,
			ID:    thid,
			Key:   thkey,
		}

		// Create first two Things with name.
		if i < 2 {
			th.Name = name
		}

		thingRepo.Save(context.Background(), th)
	}

	cases := map[string]struct {
		owner  string
		offset uint64
		limit  uint64
		name   string
		size   uint64
		total  uint64
	}{
		"retrieve all things with existing owner": {
			owner:  email,
			offset: 0,
			limit:  n,
			size:   n,
			total:  n,
		},
		"retrieve subset of things with existing owner": {
			owner:  email,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			total:  n,
		},
		"retrieve things with non-existing owner": {
			owner:  wrongValue,
			offset: 0,
			limit:  n,
			size:   0,
			total:  0,
		},
		"retrieve things with existing name": {
			owner:  email,
			offset: 1,
			limit:  n,
			name:   name,
			size:   1,
			total:  2,
		},
		"retrieve things with non-existing name": {
			owner:  email,
			offset: 0,
			limit:  n,
			name:   "wrong",
			size:   0,
			total:  0,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveAll(context.Background(), tc.owner, tc.offset, tc.limit, tc.name)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestMultiThingRetrievalByChannel(t *testing.T) {
	email := "thing-multi-retrieval-by-channel@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db)
	channelRepo := postgres.NewChannelRepository(db)

	n := uint64(10)

	chid, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cid, err := channelRepo.Save(context.Background(), things.Channel{
		ID:    chid,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	for i := uint64(0); i < n; i++ {
		thid, err := idp.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thkey, err := idp.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		th := things.Thing{
			ID:    thid,
			Owner: email,
			Key:   thkey,
		}

		tid, err := thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		err = channelRepo.Connect(context.Background(), email, cid, tid)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	}

	nonexistentChanID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner   string
		channel string
		offset  uint64
		limit   uint64
		size    uint64
		err     error
	}{
		"retrieve all things by channel with existing owner": {
			owner:   email,
			channel: cid,
			offset:  0,
			limit:   n,
			size:    n,
		},
		"retrieve subset of things by channel with existing owner": {
			owner:   email,
			channel: cid,
			offset:  n / 2,
			limit:   n,
			size:    n / 2,
		},
		"retrieve things by channel with non-existing owner": {
			owner:   wrongValue,
			channel: cid,
			offset:  0,
			limit:   n,
			size:    0,
		},
		"retrieve things by non-existing channel": {
			owner:   email,
			channel: nonexistentChanID,
			offset:  0,
			limit:   n,
			size:    0,
		},
		"retrieve things with malformed UUID": {
			owner:   email,
			channel: wrongValue,
			offset:  0,
			limit:   n,
			size:    0,
			err:     things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		page, err := thingRepo.RetrieveByChannel(context.Background(), tc.owner, tc.channel, tc.offset, tc.limit)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestThingRemoval(t *testing.T) {
	email := "thing-removal@example.com"
	thingRepo := postgres.NewThingRepository(db)

	thid, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := uuid.New().ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thing := things.Thing{
		ID:    thid,
		Owner: email,
		Key:   thkey,
	}

	id, _ := thingRepo.Save(context.Background(), thing)
	thing.ID = id

	// show that the removal works the same for both existing and non-existing
	// (removed) thing
	for i := 0; i < 2; i++ {
		err := thingRepo.Remove(context.Background(), email, thing.ID)
		require.Nil(t, err, fmt.Sprintf("#%d: failed to remove thing due to: %s", i, err))

		_, err = thingRepo.RetrieveByID(context.Background(), email, thing.ID)
		require.Equal(t, things.ErrNotFound, err, fmt.Sprintf("#%d: expected %s got %s", i, things.ErrNotFound, err))
	}
}
