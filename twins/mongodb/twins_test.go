// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/mocks"
	"github.com/absmach/magistrala/twins/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	maxNameSize = 1024
	testDB      = "test"
	collection  = "twins"
	email       = "mgx_twin@example.com"
	validName   = "mgx_twin"
	subtopic    = "engine"
)

var (
	port        string
	addr        string
	testLog, _  = mglog.New(os.Stdout, "info")
	idProvider  = uuid.New()
	invalidName = strings.Repeat("m", maxNameSize+1)
)

func TestTwinsSave(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewTwinRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentTwinID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	twin := twins.Twin{
		Owner: email,
		ID:    twid,
	}

	cases := []struct {
		desc string
		twin twins.Twin
		err  error
	}{
		{
			desc: "create new twin",
			twin: twin,
			err:  nil,
		},
		{
			desc: "create twin with invalid name",
			twin: twins.Twin{
				ID:    nonexistentTwinID,
				Owner: email,
				Name:  invalidName,
			},
			err: repoerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		_, err := repo.Save(context.Background(), tc.twin)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestTwinsUpdate(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewTwinRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentTwinID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	twin := twins.Twin{
		ID:   twid,
		Name: validName,
	}

	if _, err := repo.Save(context.Background(), twin); err != nil {
		testLog.Error(err.Error())
	}

	twin.Name = "new_name"
	cases := []struct {
		desc string
		twin twins.Twin
		err  error
	}{
		{
			desc: "update existing twin",
			twin: twin,
			err:  nil,
		},
		{
			desc: "update non-existing twin",
			twin: twins.Twin{
				ID: nonexistentTwinID,
			},
			err: repoerr.ErrNotFound,
		},
		{
			desc: "update twin with invalid name",
			twin: twins.Twin{
				ID:    twid,
				Owner: email,
				Name:  invalidName,
			},
			err: repoerr.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := repo.Update(context.Background(), tc.twin)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestTwinsRetrieveByID(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewTwinRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentTwinID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	twin := twins.Twin{
		ID: twid,
	}

	if _, err := repo.Save(context.Background(), twin); err != nil {
		testLog.Error(err.Error())
	}

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "retrieve an existing twin",
			id:   twin.ID,
			err:  nil,
		},
		{
			desc: "retrieve a non-existing twin",
			id:   nonexistentTwinID,
			err:  repoerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestTwinsRetrieveByAttribute(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewTwinRepository(db)

	chID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	empty := mocks.CreateTwin([]string{chID}, []string{""})
	_, err = repo.Save(context.Background(), empty)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	wildcard := mocks.CreateTwin([]string{chID}, []string{twins.SubtopicWildcard})
	_, err = repo.Save(context.Background(), wildcard)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	nonEmpty := mocks.CreateTwin([]string{chID}, []string{subtopic})
	_, err = repo.Save(context.Background(), nonEmpty)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		subtopic string
		ids      []string
	}{
		{
			desc:     "retrieve empty subtopic",
			subtopic: "",
			ids:      []string{wildcard.ID, empty.ID},
		},
		{
			desc:     "retrieve wildcard subtopic",
			subtopic: twins.SubtopicWildcard,
			ids:      []string{wildcard.ID},
		},
		{
			desc:     "retrieve non-empty subtopic",
			subtopic: subtopic,
			ids:      []string{wildcard.ID, nonEmpty.ID},
		},
	}

	for _, tc := range cases {
		ids, err := repo.RetrieveByAttribute(context.Background(), chID, tc.subtopic)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		assert.ElementsMatch(t, ids, tc.ids, fmt.Sprintf("%s: expected ids %v do not match received ids %v", tc.desc, tc.ids, ids))
	}
}

func TestTwinsRetrieveAll(t *testing.T) {
	email := "twin-multi-retrieval@example.com"
	name := "magistrala"
	metadata := twins.Metadata{
		"type": "test",
	}
	wrongMetadata := twins.Metadata{
		"wrong": "wrong",
	}

	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	_, err = db.Collection(collection).DeleteMany(context.Background(), bson.D{})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	twinRepo := mongodb.NewTwinRepository(db)

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		twid, err := idProvider.ID()
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		tw := twins.Twin{
			Owner:    email,
			ID:       twid,
			Metadata: metadata,
		}

		// Create first two Twins with name.
		if i < 2 {
			tw.Name = name
		}

		_, err = twinRepo.Save(context.Background(), tw)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		owner    string
		limit    uint64
		offset   uint64
		name     string
		size     uint64
		total    uint64
		metadata twins.Metadata
	}{
		"retrieve all twins with existing owner": {
			owner:  email,
			offset: 0,
			limit:  n,
			size:   n,
			total:  n,
		},
		"retrieve subset of twins with existing owner": {
			owner:  email,
			offset: 0,
			limit:  n / 2,
			size:   n / 2,
			total:  n,
		},
		"retrieve twins with non-existing owner": {
			owner:  wrongValue,
			offset: 0,
			limit:  n,
			size:   0,
			total:  0,
		},
		"retrieve twins with existing name": {
			offset: 0,
			limit:  1,
			name:   name,
			size:   1,
			total:  2,
		},
		"retrieve twins with non-existing name": {
			offset: 0,
			limit:  n,
			name:   "wrong",
			size:   0,
			total:  0,
		},
		"retrieve twins with metadata": {
			offset:   0,
			limit:    n,
			size:     n,
			total:    n,
			metadata: metadata,
		},
		"retrieve twins with wrong metadata": {
			offset:   0,
			limit:    n,
			size:     0,
			total:    0,
			metadata: wrongMetadata,
		},
	}

	for desc, tc := range cases {
		page, err := twinRepo.RetrieveAll(context.Background(), tc.owner, tc.offset, tc.limit, tc.name, tc.metadata)
		size := uint64(len(page.Twins))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestTwinsRemove(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.NewTwinRepository(db)

	twid, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentTwinID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	twin := twins.Twin{
		ID: twid,
	}

	if _, err := repo.Save(context.Background(), twin); err != nil {
		testLog.Error(err.Error())
	}

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "remove an existing twin",
			id:   twin.ID,
			err:  nil,
		},
		{
			desc: "remove a non-existing twin",
			id:   nonexistentTwinID,
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
