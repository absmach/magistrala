//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/mainflux/mainflux/things/uuid"
	"github.com/stretchr/testify/assert"
)

func TestThingSave(t *testing.T) {
	email := "thing-save@example.com"
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		Owner: email,
		Key:   uuid.New().ID(),
	}

	_, err := thingRepo.Save(thing)
	assert.Nil(t, err, fmt.Sprintf("create new thing: expected no error got %s\n", err))
}

func TestThingUpdate(t *testing.T) {
	email := "thing-update@example.com"
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		Owner: email,
		Key:   uuid.New().ID(),
	}

	id, _ := thingRepo.Save(thing)
	thing.ID = id

	cases := map[string]struct {
		thing things.Thing
		err   error
	}{
		"update existing thing":                            {thing: thing, err: nil},
		"update non-existing thing with existing user":     {thing: things.Thing{ID: wrongID, Owner: email}, err: things.ErrNotFound},
		"update existing thing ID with non-existing user":  {thing: things.Thing{ID: id, Owner: wrongValue}, err: things.ErrNotFound},
		"update non-existing thing with non-existing user": {thing: things.Thing{ID: wrongID, Owner: wrongValue}, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := thingRepo.Update(tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleThingRetrieval(t *testing.T) {
	email := "thing-single-retrieval@example.com"
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		Owner: email,
		Key:   uuid.New().ID(),
	}

	id, _ := thingRepo.Save(thing)
	thing.ID = id

	cases := map[string]struct {
		owner string
		ID    uint64
		err   error
	}{
		"retrieve thing with existing user":              {owner: thing.Owner, ID: thing.ID, err: nil},
		"retrieve non-existing thing with existing user": {owner: thing.Owner, ID: wrongID, err: things.ErrNotFound},
		"retrieve thing with non-existing owner":         {owner: wrongValue, ID: thing.ID, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := thingRepo.RetrieveByID(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRetrieveByKey(t *testing.T) {
	email := "thing-retrieved-by-key@example.com"
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		Owner: email,
		Key:   uuid.New().ID(),
	}

	id, _ := thingRepo.Save(thing)
	thing.ID = id

	cases := map[string]struct {
		key string
		ID  uint64
		err error
	}{
		"retrieve existing thing by key":     {key: thing.Key, ID: thing.ID, err: nil},
		"retrieve non-existent thing by key": {key: wrongValue, ID: wrongID, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		id, err := thingRepo.RetrieveByKey(tc.key)
		assert.Equal(t, tc.ID, id, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.ID, id))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiThingRetrieval(t *testing.T) {
	email := "thing-multi-retrieval@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)

	n := 10

	for i := 0; i < n; i++ {
		t := things.Thing{
			Owner: email,
			Key:   idp.ID(),
		}

		thingRepo.Save(t)
	}

	cases := map[string]struct {
		owner  string
		offset int
		limit  int
		size   int
	}{
		"retrieve all things with existing owner":       {owner: email, offset: 0, limit: n, size: n},
		"retrieve subset of things with existing owner": {owner: email, offset: n / 2, limit: n, size: n / 2},
		"retrieve things with non-existing owner":       {owner: wrongValue, offset: 0, limit: n, size: 0},
	}

	for desc, tc := range cases {
		n := len(thingRepo.RetrieveAll(tc.owner, tc.offset, tc.limit))
		assert.Equal(t, tc.size, n, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, n))
	}
}

func TestThingRemoval(t *testing.T) {
	email := "thing-removal@example.com"
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		Owner: email,
		Key:   uuid.New().ID(),
	}

	id, _ := thingRepo.Save(thing)
	thing.ID = id

	// show that the removal works the same for both existing and non-existing
	// (removed) thing
	for i := 0; i < 2; i++ {
		err := thingRepo.Remove(email, thing.ID)
		require.Nil(t, err, fmt.Sprintf("#%d: failed to remove thing due to: %s", i, err))

		_, err = thingRepo.RetrieveByID(email, thing.ID)
		require.Equal(t, things.ErrNotFound, err, fmt.Sprintf("#%d: expected %s got %s", i, things.ErrNotFound, err))
	}
}
