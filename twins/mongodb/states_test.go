// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestStateSave(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewStateRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	var id int64
	state := twins.State{
		TwinID:  twid,
		ID:      id,
		Created: time.Now(),
	}

	cases := []struct {
		desc  string
		state twins.State
		err   error
	}{
		{
			desc:  "save state",
			state: state,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := repo.Save(context.Background(), tc.state)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestStatesRetrieveAll(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	_, err = db.Collection("states").DeleteMany(context.Background(), bson.D{})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	repo := mongodb.NewStateRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		st := twins.State{
			TwinID:  twid,
			ID:      int64(i),
			Created: time.Now(),
		}

		err = repo.Save(context.Background(), st)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		twid   string
		limit  uint64
		offset uint64
		size   uint64
		total  uint64
	}{
		"retrieve all states with existing twin": {
			twid:   twid,
			offset: 0,
			limit:  n,
			size:   n,
			total:  n,
		},
		"retrieve subset of states with existing twin": {
			twid:   twid,
			offset: 0,
			limit:  n / 2,
			size:   n / 2,
			total:  n,
		},
		"retrieve states with non-existing twin": {
			twid:   wrongValue,
			offset: 0,
			limit:  n,
			size:   0,
			total:  0,
		},
	}

	for desc, tc := range cases {
		page, err := repo.RetrieveAll(context.Background(), tc.offset, tc.limit, tc.twid)
		size := uint64(len(page.States))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestStatesRetrieveLast(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	_, err = db.Collection("states").DeleteMany(context.Background(), bson.D{})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	repo := mongodb.NewStateRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	n := int64(10)
	for i := int64(1); i <= n; i++ {
		st := twins.State{
			TwinID:  twid,
			ID:      i,
			Created: time.Now(),
		}

		err = repo.Save(context.Background(), st)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		twid string
		id   int64
	}{
		"retrieve last state with existing twin": {
			twid: twid,
			id:   n,
		},
		"retrieve states with non-existing owner": {
			twid: wrongValue,
			id:   0,
		},
	}

	for desc, tc := range cases {
		state, err := repo.RetrieveLast(context.Background(), tc.twid)
		assert.Equal(t, tc.id, state.ID, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.id, state.ID))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}
