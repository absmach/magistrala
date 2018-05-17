package postgres_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/postgres"
	"github.com/mainflux/mainflux/things/uuid"
	"github.com/stretchr/testify/assert"
)

func TestThingSave(t *testing.T) {
	email := "thing-save@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)
	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}

	_, err := thingRepo.Save(thing)
	hasErr := err != nil

	assert.False(t, hasErr, fmt.Sprintf("create new thing: expected false got %t\n", hasErr))
}

func TestThingUpdate(t *testing.T) {
	email := "thing-update@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}

	thingRepo.Save(thing)

	cases := map[string]struct {
		thing things.Thing
		err   error
	}{
		"existing thing":                            {thing, nil},
		"non-existing thing with existing user":     {things.Thing{ID: wrong, Owner: email}, things.ErrNotFound},
		"non-existing thing with non-existing user": {things.Thing{ID: wrong, Owner: wrong}, things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := thingRepo.Update(tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestSingleThingRetrieval(t *testing.T) {
	email := "thing-single-retrieval@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)

	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}

	thingRepo.Save(thing)

	cases := map[string]struct {
		owner string
		ID    string
		err   error
	}{
		"existing user":                     {thing.Owner, thing.ID, nil},
		"existing user, non-existing thing": {thing.Owner, wrong, things.ErrNotFound},
		"non-existing owner":                {wrong, thing.ID, things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := thingRepo.RetrieveByID(tc.owner, tc.ID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestThingRetrieveByKey(t *testing.T) {
	email := "thing-retrieved-by-key@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)
	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}
	thingRepo.Save(thing)

	cases := map[string]struct {
		key string
		id  string
		err error
	}{
		"retrieve existing thing by key":     {thing.Key, thing.ID, nil},
		"retrieve non-existent thing by key": {wrong, "", things.ErrNotFound},
	}

	for desc, tc := range cases {
		id, err := thingRepo.RetrieveByKey(tc.key)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
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
			ID:    idp.ID(),
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
		"existing owner, retrieve all":    {email, 0, n, n},
		"existing owner, retrieve subset": {email, 1, 6, 6},
		"non-existing owner":              {wrong, 1, 6, 0},
	}

	for desc, tc := range cases {
		n := len(thingRepo.RetrieveAll(tc.owner, tc.offset, tc.limit))
		assert.Equal(t, tc.size, n, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, n))
	}
}

func TestThingRemoval(t *testing.T) {
	email := "thing-removal@example.com"
	idp := uuid.New()
	thingRepo := postgres.NewThingRepository(db, testLog)
	thing := things.Thing{
		ID:    idp.ID(),
		Owner: email,
		Key:   idp.ID(),
	}
	thingRepo.Save(thing)

	// show that the removal works the same for both existing and non-existing
	// (removed) thing
	for i := 0; i < 2; i++ {
		if err := thingRepo.Remove(email, thing.ID); err != nil {
			t.Fatalf("#%d: failed to remove thing due to: %s", i, err)
		}

		if _, err := thingRepo.RetrieveByID(email, thing.ID); err != things.ErrNotFound {
			t.Fatalf("#%d: expected %s got %s", i, things.ErrNotFound, err)
		}
	}
}
