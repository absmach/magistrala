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

	"github.com/gofrs/uuid"
	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConfigs = 10

var (
	config = bootstrap.Config{
		MFThing:     "mf-thing",
		MFKey:       "mf-key",
		ExternalID:  "external-id",
		ExternalKey: "external-key",
		Owner:       "user@email.com",
		MFChannels: []bootstrap.Channel{
			bootstrap.Channel{ID: "1", Name: "name 1", Metadata: map[string]interface{}{"meta": 1.0}},
			bootstrap.Channel{ID: "2", Name: "name 2", Metadata: map[string]interface{}{"meta": 2.0}},
		},
		Content: "content",
		State:   bootstrap.Inactive,
	}

	channels = []string{"1", "2"}
)

func TestSave(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	diff := "different"

	duplicateThing := config
	duplicateThing.ExternalID = diff
	duplicateThing.MFKey = diff
	duplicateThing.MFChannels = []bootstrap.Channel{}

	duplicateExternal := config
	duplicateExternal.MFThing = diff
	duplicateExternal.MFKey = diff
	duplicateExternal.MFChannels = []bootstrap.Channel{}

	duplicateChannels := config
	duplicateChannels.ExternalID = diff
	duplicateChannels.MFKey = diff
	duplicateChannels.MFThing = diff

	cases := []struct {
		desc        string
		config      bootstrap.Config
		connections []string
		err         error
	}{
		{
			desc:        "save a config",
			config:      config,
			connections: channels,
			err:         nil,
		},
		{
			desc:        "save config with same Thing ID",
			config:      duplicateThing,
			connections: nil,
			err:         bootstrap.ErrConflict,
		},
		{
			desc:        "save config with same external ID",
			config:      duplicateExternal,
			connections: nil,
			err:         bootstrap.ErrConflict,
		},
		{
			desc:        "save config with same Channels",
			config:      duplicateChannels,
			connections: channels,
			err:         bootstrap.ErrConflict,
		},
	}
	for _, tc := range cases {
		_, err := repo.Save(tc.config, tc.connections)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	nonexistentConfID, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))

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
			id:    nonexistentConfID.String(),
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
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	for i := 0; i < numConfigs; i++ {
		c := config
		// Use UUID to prevent conflict errors.

		uid, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = uid.String()
		c.Name = fmt.Sprintf("name %d", i)
		c.MFThing = uid.String()
		c.MFKey = uid.String()

		if i%2 == 0 {
			c.State = bootstrap.Active
		}

		if i > 0 {
			c.MFChannels = nil
		}

		_, err = repo.Save(c, channels)
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
		size := len(ret.Configs)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRetrieveByExternalID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc       string
		externalID string
		err        error
	}{
		{
			desc:       "retrieve with invalid external ID",
			externalID: strconv.Itoa(numConfigs + 1),
			err:        bootstrap.ErrNotFound,
		},
		{
			desc:       "retrieve with external key",
			externalID: c.ExternalID,
			err:        nil,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByExternalID(tc.externalID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	c.Content = "new content"
	c.Name = "new name"

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

func TestUpdateCert(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	c.Content = "new content"
	c.Name = "new name"

	wrongOwner := c
	wrongOwner.Owner = "3"

	cases := []struct {
		desc    string
		key     string
		owner   string
		cert    string
		certKey string
		ca      string
		err     error
	}{
		{
			desc:    "update with wrong owner",
			key:     "",
			cert:    "cert",
			certKey: "certKey",
			ca:      "",
			owner:   "wrong",
			err:     bootstrap.ErrNotFound,
		},
		{
			desc:    "update a config",
			key:     c.MFKey,
			cert:    "cert",
			certKey: "certKey",
			ca:      "ca",
			owner:   c.Owner,
			err:     nil,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateCert(tc.owner, tc.key, tc.cert, tc.key, tc.ca)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateConnections(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	// Use UUID to prevent conflicts.
	uid, err = uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	c.MFChannels = []bootstrap.Channel{}
	c2, err := repo.Save(c, []string{channels[0]})
	require.Nil(t, err, fmt.Sprintf("Saving a config expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		key         string
		id          string
		channels    []bootstrap.Channel
		connections []string
		err         error
	}{
		{
			desc:        "update connections of non-existing config",
			key:         config.Owner,
			id:          "unknown",
			channels:    nil,
			connections: []string{channels[1]},
			err:         bootstrap.ErrNotFound,
		},
		{
			desc:        "update connections",
			key:         config.Owner,
			id:          c.MFThing,
			channels:    nil,
			connections: []string{channels[1]},
			err:         nil,
		},
		{
			desc:        "update connections with existing channels",
			key:         config.Owner,
			id:          c2,
			channels:    nil,
			connections: channels,
			err:         nil,
		},
		{
			desc:        "update connections no channels",
			key:         config.Owner,
			id:          c.MFThing,
			channels:    nil,
			connections: nil,
			err:         nil,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateConnections(tc.key, tc.id, tc.channels, tc.connections)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(c, channels)
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
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(c, channels)
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
			id:    "wrong",
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

func TestListExisting(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	var chs []bootstrap.Channel
	for _, ch := range config.MFChannels {
		chs = append(chs, ch)
	}

	cases := []struct {
		desc        string
		key         string
		connections []string
		existing    []bootstrap.Channel
	}{
		{
			desc:        "list all existing channels",
			key:         c.Owner,
			connections: channels,
			existing:    chs,
		},
		{
			desc:        "list a subset of existing channels",
			key:         c.Owner,
			connections: []string{channels[0], "5"},
			existing:    []bootstrap.Channel{chs[0]},
		},
		{
			desc:        "list a subset of existing channels empty",
			key:         c.Owner,
			connections: []string{"5", "6"},
			existing:    []bootstrap.Channel{},
		},
	}
	for _, tc := range cases {
		existing, err := repo.ListExisting(tc.key, tc.connections)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.ElementsMatch(t, tc.existing, existing, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.existing, existing))
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
			externalID:  "unknown",
			externalKey: "unknown",
			err:         nil,
		},
		{
			desc:        "save invalid unknown",
			externalID:  "unknown",
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
		id, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		repo.SaveUnknown(id.String(), id.String())
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
		size := len(ret.Configs)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRemoveThing(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	for i := 0; i < 2; i++ {
		err := repo.RemoveThing(saved)
		assert.Nil(t, err, fmt.Sprintf("an unexpected error occured: %s\n", err))
	}
}

func TestUpdateChannel(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	id := c.MFChannels[0].ID
	update := bootstrap.Channel{
		ID:       id,
		Name:     "update name",
		Metadata: map[string]interface{}{"update": "metadata update"},
	}
	err = repo.UpdateChannel(update)
	assert.Nil(t, err, fmt.Sprintf("updating config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(c.Owner, c.MFThing)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	var retreved bootstrap.Channel
	for _, c := range cfg.MFChannels {
		if c.ID == id {
			retreved = c
			break
		}
	}

	assert.Equal(t, update, retreved, fmt.Sprintf("expected %s, go %s", update, retreved))
}

func TestRemoveChannel(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.RemoveChannel(c.MFChannels[0].ID)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(c.Owner, c.MFThing)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.NotContains(t, cfg.MFChannels, c.MFChannels[0], fmt.Sprintf("expected to remove channel %s from %s", c.MFChannels[0], cfg.MFChannels))
}

func TestDisconnectThing(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.MFKey = uid.String()
	c.MFThing = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.DisconnectThing(c.MFChannels[0].ID, saved)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(c.Owner, c.MFThing)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.Equal(t, cfg.State, bootstrap.Inactive, fmt.Sprintf("expected ti be inactive when a connection is removed from %s", cfg))
}

func deleteChannels(repo bootstrap.ConfigRepository) error {
	for _, ch := range channels {
		if err := repo.RemoveChannel(ch); err != nil {
			return err
		}
	}

	return nil
}
