// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/postgres"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConfigs = 10

var (
	config = bootstrap.Config{
		ThingID:     "mg-thing",
		ThingKey:    "mg-key",
		ExternalID:  "external-id",
		ExternalKey: "external-key",
		Owner:       "user@email.com",
		Channels: []bootstrap.Channel{
			{ID: "1", Name: "name 1", Metadata: map[string]interface{}{"meta": 1.0}},
			{ID: "2", Name: "name 2", Metadata: map[string]interface{}{"meta": 2.0}},
		},
		Content: "content",
		State:   bootstrap.Inactive,
	}

	channels = []string{"1", "2"}
)

func TestSave(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	diff := "different"

	duplicateThing := config
	duplicateThing.ExternalID = diff
	duplicateThing.ThingKey = diff
	duplicateThing.Channels = []bootstrap.Channel{}

	duplicateExternal := config
	duplicateExternal.ThingID = diff
	duplicateExternal.ThingKey = diff
	duplicateExternal.Channels = []bootstrap.Channel{}

	duplicateChannels := config
	duplicateChannels.ExternalID = diff
	duplicateChannels.ThingKey = diff
	duplicateChannels.ThingID = diff

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
			err:         errors.ErrConflict,
		},
		{
			desc:        "save config with same external ID",
			config:      duplicateExternal,
			connections: nil,
			err:         errors.ErrConflict,
		},
		{
			desc:        "save config with same Channels",
			config:      duplicateChannels,
			connections: channels,
			err:         errors.ErrConflict,
		},
	}
	for _, tc := range cases {
		id, err := repo.Save(context.Background(), tc.config, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, id, tc.config.ThingID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.config.ThingID, id))
		}
	}
}

func TestRetrieveByID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c, channels)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "retrieve a non-existing config",
			owner: c.Owner,
			id:    nonexistentConfID.String(),
			err:   errors.ErrNotFound,
		},
		{
			desc:  "retrieve a config with invalid ID",
			owner: c.Owner,
			id:    "invalid",
			err:   errors.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.owner, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	for i := 0; i < numConfigs; i++ {
		c := config

		// Use UUID to prevent conflict errors.
		uid, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = uid.String()
		c.Name = fmt.Sprintf("name %d", i)
		c.ThingID = uid.String()
		c.ThingKey = uid.String()

		if i%2 == 0 {
			c.State = bootstrap.Active
		}

		if i > 0 {
			c.Channels = nil
		}

		_, err = repo.Save(context.Background(), c, channels)
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
		ret := repo.RetrieveAll(context.Background(), tc.owner, tc.filter, tc.offset, tc.limit)
		size := len(ret.Configs)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRetrieveByExternalID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc       string
		externalID string
		err        error
	}{
		{
			desc:       "retrieve with invalid external ID",
			externalID: strconv.Itoa(numConfigs + 1),
			err:        errors.ErrNotFound,
		},
		{
			desc:       "retrieve with external key",
			externalID: c.ExternalID,
			err:        nil,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByExternalID(context.Background(), tc.externalID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdate(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

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
			err:    errors.ErrNotFound,
		},
		{
			desc:   "update a config",
			config: c,
			err:    nil,
		},
	}
	for _, tc := range cases {
		err := repo.Update(context.Background(), tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateCert(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	c.Content = "new content"
	c.Name = "new name"

	wrongOwner := c
	wrongOwner.Owner = "3"

	cases := []struct {
		desc           string
		thingID        string
		owner          string
		cert           string
		certKey        string
		ca             string
		expectedConfig bootstrap.Config
		err            error
	}{
		{
			desc:           "update with wrong owner",
			thingID:        "",
			cert:           "cert",
			certKey:        "certKey",
			ca:             "",
			owner:          "wrong",
			expectedConfig: bootstrap.Config{},
			err:            errors.ErrNotFound,
		},
		{
			desc:    "update a config",
			thingID: c.ThingID,
			cert:    "cert",
			certKey: "certKey",
			ca:      "ca",
			owner:   c.Owner,
			expectedConfig: bootstrap.Config{
				ThingID:    c.ThingID,
				ClientCert: "cert",
				CACert:     "ca",
				ClientKey:  "certKey",
				Owner:      c.Owner,
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		cfg, err := repo.UpdateCert(context.Background(), tc.owner, tc.thingID, tc.cert, tc.certKey, tc.ca)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.expectedConfig, cfg))
	}
}

func TestUpdateConnections(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	// Use UUID to prevent conflicts.
	uid, err = uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	c.Channels = []bootstrap.Channel{}
	c2, err := repo.Save(context.Background(), c, []string{channels[0]})
	assert.Nil(t, err, fmt.Sprintf("Saving a config expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		owner       string
		id          string
		channels    []bootstrap.Channel
		connections []string
		err         error
	}{
		{
			desc:        "update connections of non-existing config",
			owner:       config.Owner,
			id:          "unknown",
			channels:    nil,
			connections: []string{channels[1]},
			err:         errors.ErrNotFound,
		},
		{
			desc:        "update connections",
			owner:       config.Owner,
			id:          c.ThingID,
			channels:    nil,
			connections: []string{channels[1]},
			err:         nil,
		},
		{
			desc:        "update connections with existing channels",
			owner:       config.Owner,
			id:          c2,
			channels:    nil,
			connections: channels,
			err:         nil,
		},
		{
			desc:        "update connections no channels",
			owner:       config.Owner,
			id:          c.ThingID,
			channels:    nil,
			connections: nil,
			err:         nil,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateConnections(context.Background(), tc.owner, tc.id, tc.channels, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	// Removal works the same for both existing and non-existing
	// (removed) config
	for i := 0; i < 2; i++ {
		err := repo.Remove(context.Background(), c.Owner, id)
		assert.Nil(t, err, fmt.Sprintf("%d: failed to remove config due to: %s", i, err))

		_, err = repo.RetrieveByID(context.Background(), c.Owner, id)
		assert.True(t, errors.Contains(err, errors.ErrNotFound), fmt.Sprintf("%d: expected %s got %s", i, errors.ErrNotFound, err))
	}
}

func TestChangeState(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "change state with wrong id",
			id:    "wrong",
			owner: c.Owner,
			err:   errors.ErrNotFound,
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
		err := repo.ChangeState(context.Background(), tc.owner, tc.id, tc.state)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListExisting(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	var chs []bootstrap.Channel
	chs = append(chs, config.Channels...)

	cases := []struct {
		desc        string
		owner       string
		connections []string
		existing    []bootstrap.Channel
	}{
		{
			desc:        "list all existing channels",
			owner:       c.Owner,
			connections: channels,
			existing:    chs,
		},
		{
			desc:        "list a subset of existing channels",
			owner:       c.Owner,
			connections: []string{channels[0], "5"},
			existing:    []bootstrap.Channel{chs[0]},
		},
		{
			desc:        "list a subset of existing channels empty",
			owner:       c.Owner,
			connections: []string{"5", "6"},
			existing:    []bootstrap.Channel{},
		},
	}
	for _, tc := range cases {
		existing, err := repo.ListExisting(context.Background(), tc.owner, tc.connections)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.ElementsMatch(t, tc.existing, existing, fmt.Sprintf("%s: Got non-matching elements.", tc.desc))
	}
}

func TestRemoveThing(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	for i := 0; i < 2; i++ {
		err := repo.RemoveThing(context.Background(), saved)
		assert.Nil(t, err, fmt.Sprintf("an unexpected error occurred: %s\n", err))
	}
}

func TestUpdateChannel(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	id := c.Channels[0].ID
	update := bootstrap.Channel{
		ID:       id,
		Name:     "update name",
		Metadata: map[string]interface{}{"update": "metadata update"},
	}
	err = repo.UpdateChannel(context.Background(), update)
	assert.Nil(t, err, fmt.Sprintf("updating config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(context.Background(), c.Owner, c.ThingID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	var retreved bootstrap.Channel
	for _, c := range cfg.Channels {
		if c.ID == id {
			retreved = c
			break
		}
	}
	update.Owner = retreved.Owner
	assert.Equal(t, update, retreved, fmt.Sprintf("expected %s, go %s", update, retreved))
}

func TestRemoveChannel(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.RemoveChannel(context.Background(), c.Channels[0].ID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(context.Background(), c.Owner, c.ThingID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.NotContains(t, cfg.Channels, c.Channels[0], fmt.Sprintf("expected to remove channel %s from %s", c.Channels[0], cfg.Channels))
}

func TestDisconnectThing(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ThingKey = uid.String()
	c.ThingID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.DisconnectThing(context.Background(), c.Channels[0].ID, saved)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(context.Background(), c.Owner, c.ThingID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.Equal(t, cfg.State, bootstrap.Inactive, fmt.Sprintf("expected ti be inactive when a connection is removed from %s", cfg))
}

func deleteChannels(ctx context.Context, repo bootstrap.ConfigRepository) error {
	for _, ch := range channels {
		if err := repo.RemoveChannel(ctx, ch); err != nil {
			return err
		}
	}

	return nil
}
