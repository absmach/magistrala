//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package postgres_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/postgres"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConfigs = 10

var config = bootstrap.Config{
	MFThing:     "mf-thing",
	MFKey:       "mf-key",
	ExternalID:  "external-id",
	ExternalKey: "external-key",
	Owner:       "user@email.com",
	MFChannels: []bootstrap.Channel{
		bootstrap.Channel{ID: "1", Name: "name 1", Metadata: "{\"meta\":1}"},
		bootstrap.Channel{ID: "2", Name: "name 2", Metadata: "{\"meta\":2}"},
	},
	Content: "content",
	State:   bootstrap.Inactive,
}

func TestSave(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	cases := []struct {
		desc   string
		config bootstrap.Config
		err    error
	}{
		{
			desc:   "save a config",
			config: config,
			err:    nil,
		},
		{
			desc:   "save config with same external ID",
			config: config,
			err:    bootstrap.ErrConflict,
		},
	}
	for _, tc := range cases {
		_, err := repo.Save(tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	c := config
	// Use UUID to prevent conflicts.
	id := uuid.NewV4().String()
	c.MFKey = id
	c.MFThing = id
	c.ExternalID = id
	c.ExternalKey = id
	id, err := repo.Save(c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	cases := []struct {
		desc  string
		owner string
		id    string
		err   error
	}{
		{
			desc:  "retrieve config",
			owner: c.Owner,
			id:    id,
			err:   nil,
		},
		{
			desc:  "retrieve config with wrong owner",
			owner: "2",
			id:    id,
			err:   bootstrap.ErrNotFound,
		},
		{
			desc:  "retrieve a non-existing config",
			owner: c.Owner,
			id:    uuid.NewV4().String(),
			err:   bootstrap.ErrNotFound,
		},
		{
			desc:  "retrieve a config with invalid ID",
			owner: c.Owner,
			id:    "invalid",
			err:   bootstrap.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByID(tc.owner, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
func TestRetrieveAll(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	for i := 0; i < numConfigs; i++ {
		c := config
		// Use UUID to prevent conflict errors.
		id := uuid.NewV4().String()
		c.ExternalID = id
		c.Name = fmt.Sprintf("name %d", i)
		c.MFThing = id
		c.MFKey = id
		if i%2 == 0 {
			c.State = bootstrap.Active
		}
		_, err := repo.Save(c)
		require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	}
	cases := []struct {
		desc   string
		owner  string
		offset uint64
		limit  uint64
		filter bootstrap.Filter
		size   int
	}{
		{
			desc:   "retrieve all",
			owner:  config.Owner,
			offset: 0,
			limit:  uint64(numConfigs),
			size:   numConfigs,
		},
		{
			desc:   "retrieve subset",
			owner:  config.Owner,
			offset: 5,
			limit:  uint64(numConfigs - 5),
			size:   numConfigs - 5,
		},
		{
			desc:   "retrieve wrong owner",
			owner:  "2",
			offset: 0,
			limit:  uint64(numConfigs),
			size:   0,
		},
		{
			desc:   "retrieve all active",
			owner:  config.Owner,
			offset: 0,
			limit:  uint64(numConfigs),
			filter: bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			size:   numConfigs / 2,
		},
		{
			desc:   "retrieve search by name",
			owner:  config.Owner,
			offset: 0,
			limit:  uint64(numConfigs),
			filter: bootstrap.Filter{PartialMatch: map[string]string{"name": "1"}},
			size:   1,
		},
	}
	for _, tc := range cases {
		ret := repo.RetrieveAll(tc.owner, tc.filter, tc.offset, tc.limit)
		size := len(ret)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRetrieveByExternalID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	c := config
	// Use UUID to prevent conflicts.
	id := uuid.NewV4().String()
	c.MFKey = id
	c.MFThing = id
	c.ExternalID = id
	c.ExternalKey = id
	_, err := repo.Save(c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		err         error
	}{
		{
			desc:        "retrieve with invalid external ID",
			externalID:  strconv.Itoa(numConfigs + 1),
			externalKey: config.ExternalKey,
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "retrieve with invalid external key",
			externalID:  c.ExternalID,
			externalKey: "invalid",
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "retrieve with external key",
			externalID:  c.ExternalID,
			externalKey: c.ExternalKey,
			err:         nil,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByExternalID(tc.externalKey, tc.externalID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	c := config
	// Use UUID to prevent conflicts.
	id := uuid.NewV4().String()
	c.MFKey = id
	c.MFThing = id
	c.ExternalID = id
	c.ExternalKey = id
	saved, err := repo.Save(c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	id = uuid.NewV4().String()
	c.MFThing = saved
	c.ExternalID = id
	c.ExternalKey = id
	c.MFChannels = append(config.MFChannels, bootstrap.Channel{ID: "3", Name: "name 3", Metadata: `{"meta": 3}`})
	c.State = bootstrap.Active
	c.Content = "new content"

	wrongOwner := c
	wrongOwner.Owner = "3"

	cases := []struct {
		desc   string
		id     string
		config bootstrap.Config
		err    error
	}{
		{
			desc:   "update with wrong owner",
			config: wrongOwner,
			err:    bootstrap.ErrNotFound,
		},
		{
			desc:   "update a config",
			config: c,
			err:    nil,
		},
	}
	for _, tc := range cases {
		err := repo.Update(tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	c := config
	// Use UUID to prevent conflicts.
	id := uuid.NewV4().String()
	c.MFKey = id
	c.MFThing = id
	c.ExternalID = id
	c.ExternalKey = id
	id, err := repo.Save(c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	// Removal works the same for both existing and non-existing
	// (removed) config
	for i := 0; i < 2; i++ {
		err := repo.Remove(c.Owner, id)
		require.Nil(t, err, fmt.Sprintf("%d: failed to remove config due to: %s", i, err))

		_, err = repo.RetrieveByID(c.Owner, id)
		require.Equal(t, bootstrap.ErrNotFound, err, fmt.Sprintf("%d: expected %s got %s", i, bootstrap.ErrNotFound, err))
	}
}

func TestChangeState(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	c := config
	// Use UUID to prevent conflicts.
	id := uuid.NewV4().String()
	c.MFKey = id
	c.MFThing = id
	c.ExternalID = id
	c.ExternalKey = id
	saved, err := repo.Save(c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc  string
		owner string
		id    string
		state bootstrap.State
		err   error
	}{
		{
			desc:  "change state with wrong owner",
			id:    saved,
			owner: "2",
			err:   bootstrap.ErrNotFound,
		},
		{
			desc:  "change state with wrong id",
			id:    uuid.NewV4().String(),
			owner: c.Owner,
			err:   bootstrap.ErrNotFound,
		},
		{
			desc:  "change state to Active",
			id:    saved,
			owner: c.Owner,
			state: bootstrap.Active,
			err:   nil,
		},
		{
			desc:  "change state to Inactive",
			id:    saved,
			owner: c.Owner,
			state: bootstrap.Inactive,
			err:   nil,
		},
	}
	for _, tc := range cases {
		err := repo.ChangeState(tc.owner, tc.id, tc.state)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSaveUnknown(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		err         error
	}{
		{
			desc:        "save unknown",
			externalID:  uuid.NewV4().String(),
			externalKey: uuid.NewV4().String(),
			err:         nil,
		},
		{
			desc:        "save invalid unknown",
			externalID:  uuid.NewV4().String(),
			externalKey: "",
			err:         nil,
		},
	}
	for _, tc := range cases {
		err := repo.SaveUnknown(tc.externalKey, tc.externalID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveUnknown(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	for i := 0; i < numConfigs; i++ {
		id := uuid.NewV4().String()
		repo.SaveUnknown(id, id)
	}

	cases := []struct {
		desc   string
		offset uint64
		limit  uint64
		size   int
	}{
		{
			desc:   "retrieve all",
			offset: 0,
			limit:  uint64(numConfigs),
			size:   numConfigs,
		},
		{
			desc:   "retrieve a subset",
			offset: 5,
			limit:  uint64(numConfigs - 5),
			size:   numConfigs - 5,
		},
	}
	for _, tc := range cases {
		ret := repo.RetrieveUnknown(tc.offset, tc.limit)
		size := len(ret)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRemoveUnknown(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	id := uuid.NewV4().String()
	repo.SaveUnknown(id, id)

	// Removal works the same for both existing and non-existing
	// (removed) config
	for i := 0; i < 2; i++ {
		err := repo.RemoveUnknown(id, id)
		require.Nil(t, err, fmt.Sprintf("%d: failed to remove config due to: %s", i, err))
	}
}
