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
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConfigs = 10

var (
	config = bootstrap.Config{
		ClientID:     "mg-client",
		ClientSecret: "mg-key",
		ExternalID:   "external-id",
		ExternalKey:  "external-key",
		DomainID:     testsutil.GenerateUUID(&testing.T{}),
		Channels: []bootstrap.Channel{
			{ID: "1", Name: "name 1", Metadata: map[string]any{"meta": 1.0}},
			{ID: "2", Name: "name 2", Metadata: map[string]any{"meta": 2.0}},
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

	duplicateClient := config
	duplicateClient.ExternalID = diff
	duplicateClient.ClientSecret = diff
	duplicateClient.Channels = []bootstrap.Channel{}

	duplicateExternal := config
	duplicateExternal.ClientID = diff
	duplicateExternal.ClientSecret = diff
	duplicateExternal.Channels = []bootstrap.Channel{}

	duplicateChannels := config
	duplicateChannels.ExternalID = diff
	duplicateChannels.ClientSecret = diff
	duplicateChannels.ClientID = diff

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
			desc:        "save config with same Client ID",
			config:      duplicateClient,
			connections: nil,
			err:         repoerr.ErrConflict,
		},
		{
			desc:        "save config with same external ID",
			config:      duplicateExternal,
			connections: nil,
			err:         repoerr.ErrConflict,
		},
		{
			desc:        "save config with same Channels",
			config:      duplicateChannels,
			connections: channels,
			err:         repoerr.ErrConflict,
		},
	}
	for _, tc := range cases {
		id, err := repo.Save(context.Background(), tc.config, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, id, tc.config.ClientID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.config.ClientID, id))
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c, channels)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	nonexistentConfID, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))

	cases := []struct {
		desc     string
		domainID string
		id       string
		err      error
	}{
		{
			desc:     "retrieve config",
			domainID: c.DomainID,
			id:       id,
			err:      nil,
		},
		{
			desc:     "retrieve config with wrong domain ID ",
			domainID: "2",
			id:       id,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve a non-existing config",
			domainID: c.DomainID,
			id:       nonexistentConfID.String(),
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "retrieve a config with invalid ID",
			domainID: c.DomainID,
			id:       "invalid",
			err:      repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.domainID, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	clientIDs := make([]string, numConfigs)

	for i := 0; i < numConfigs; i++ {
		c := config

		// Use UUID to prevent conflict errors.
		uid, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = uid.String()
		c.Name = fmt.Sprintf("name %d", i)
		c.ClientID = uid.String()
		c.ClientSecret = uid.String()

		clientIDs[i] = c.ClientID

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
		desc     string
		domainID string
		clientID []string
		offset   uint64
		limit    uint64
		filter   bootstrap.Filter
		size     int
	}{
		{
			desc:     "retrieve all configs",
			domainID: config.DomainID,
			clientID: []string{},
			offset:   0,
			limit:    uint64(numConfigs),
			size:     numConfigs,
		},
		{
			desc:     "retrieve a subset of configs",
			domainID: config.DomainID,
			clientID: []string{},
			offset:   5,
			limit:    uint64(numConfigs - 5),
			size:     numConfigs - 5,
		},
		{
			desc:     "retrieve with wrong domain ID ",
			domainID: "2",
			clientID: []string{},
			offset:   0,
			limit:    uint64(numConfigs),
			size:     0,
		},
		{
			desc:     "retrieve all active configs ",
			domainID: config.DomainID,
			clientID: []string{},
			offset:   0,
			limit:    uint64(numConfigs),
			filter:   bootstrap.Filter{FullMatch: map[string]string{"state": bootstrap.Active.String()}},
			size:     numConfigs / 2,
		},
		{
			desc:     "retrieve all with partial match filter",
			domainID: config.DomainID,
			clientID: []string{},
			offset:   0,
			limit:    uint64(numConfigs),
			filter:   bootstrap.Filter{PartialMatch: map[string]string{"name": "1"}},
			size:     1,
		},
		{
			desc:     "retrieve search by name",
			domainID: config.DomainID,
			clientID: []string{},
			offset:   0,
			limit:    uint64(numConfigs),
			filter:   bootstrap.Filter{PartialMatch: map[string]string{"name": "1"}},
			size:     1,
		},
		{
			desc:     "retrieve by valid clientIDs",
			domainID: config.DomainID,
			clientID: clientIDs,
			offset:   0,
			limit:    uint64(numConfigs),
			size:     10,
		},
		{
			desc:     "retrieve by non-existing clientID",
			domainID: config.DomainID,
			clientID: []string{"non-existing"},
			offset:   0,
			limit:    uint64(numConfigs),
			size:     0,
		},
	}
	for _, tc := range cases {
		ret := repo.RetrieveAll(context.Background(), tc.domainID, tc.clientID, tc.filter, tc.offset, tc.limit)
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
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
			err:        repoerr.ErrNotFound,
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	c.Content = "new content"
	c.Name = "new name"

	wrongDomainID := c
	wrongDomainID.DomainID = "3"

	cases := []struct {
		desc   string
		id     string
		config bootstrap.Config
		err    error
	}{
		{
			desc:   "update with wrong domainID ",
			config: wrongDomainID,
			err:    repoerr.ErrNotFound,
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	c.Content = "new content"
	c.Name = "new name"

	wrongDomainID := c
	wrongDomainID.DomainID = "3"

	cases := []struct {
		desc           string
		clientID       string
		domainID       string
		cert           string
		certKey        string
		ca             string
		expectedConfig bootstrap.Config
		err            error
	}{
		{
			desc:           "update with wrong domain ID ",
			clientID:       "",
			cert:           "cert",
			certKey:        "certKey",
			ca:             "",
			domainID:       wrongDomainID.DomainID,
			expectedConfig: bootstrap.Config{},
			err:            repoerr.ErrNotFound,
		},
		{
			desc:     "update a config",
			clientID: c.ClientID,
			cert:     "cert",
			certKey:  "certKey",
			ca:       "ca",
			domainID: c.DomainID,
			expectedConfig: bootstrap.Config{
				ClientID:   c.ClientID,
				ClientCert: "cert",
				CACert:     "ca",
				ClientKey:  "certKey",
				DomainID:   c.DomainID,
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		cfg, err := repo.UpdateCert(context.Background(), tc.domainID, tc.clientID, tc.cert, tc.certKey, tc.ca)
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	// Use UUID to prevent conflicts.
	uid, err = uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	c.Channels = []bootstrap.Channel{}
	c2, err := repo.Save(context.Background(), c, []string{channels[0]})
	assert.Nil(t, err, fmt.Sprintf("Saving a config expected to succeed: %s.\n", err))

	cases := []struct {
		desc        string
		domainID    string
		id          string
		channels    []bootstrap.Channel
		connections []string
		err         error
	}{
		{
			desc:        "update connections of non-existing config",
			domainID:    config.DomainID,
			id:          "unknown",
			channels:    nil,
			connections: []string{channels[1]},
			err:         repoerr.ErrNotFound,
		},
		{
			desc:        "update connections",
			domainID:    config.DomainID,
			id:          c.ClientID,
			channels:    nil,
			connections: []string{channels[1]},
			err:         nil,
		},
		{
			desc:        "update connections with existing channels",
			domainID:    config.DomainID,
			id:          c2,
			channels:    nil,
			connections: channels,
			err:         nil,
		},
		{
			desc:        "update connections no channels",
			domainID:    config.DomainID,
			id:          c.ClientID,
			channels:    nil,
			connections: nil,
			err:         nil,
		},
	}
	for _, tc := range cases {
		err := repo.UpdateConnections(context.Background(), tc.domainID, tc.id, tc.channels, tc.connections)
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	// Removal works the same for both existing and non-existing
	// (removed) config
	for i := 0; i < 2; i++ {
		err := repo.Remove(context.Background(), c.DomainID, id)
		assert.Nil(t, err, fmt.Sprintf("%d: failed to remove config due to: %s", i, err))

		_, err = repo.RetrieveByID(context.Background(), c.DomainID, id)
		assert.True(t, errors.Contains(err, repoerr.ErrNotFound), fmt.Sprintf("%d: expected %s got %s", i, repoerr.ErrNotFound, err))
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc     string
		domainID string
		id       string
		state    bootstrap.State
		err      error
	}{
		{
			desc:     "change state with wrong domain ID ",
			id:       saved,
			domainID: "2",
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "change state with wrong id",
			id:       "wrong",
			domainID: c.DomainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "change state to Active",
			id:       saved,
			domainID: c.DomainID,
			state:    bootstrap.Active,
			err:      nil,
		},
		{
			desc:     "change state to Inactive",
			id:       saved,
			domainID: c.DomainID,
			state:    bootstrap.Inactive,
			err:      nil,
		},
	}
	for _, tc := range cases {
		err := repo.ChangeState(context.Background(), tc.domainID, tc.id, tc.state)
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	var chs []bootstrap.Channel
	chs = append(chs, config.Channels...)

	cases := []struct {
		desc        string
		domainID    string
		connections []string
		existing    []bootstrap.Channel
	}{
		{
			desc:        "list all existing channels",
			domainID:    c.DomainID,
			connections: channels,
			existing:    chs,
		},
		{
			desc:        "list a subset of existing channels",
			domainID:    c.DomainID,
			connections: []string{channels[0], "5"},
			existing:    []bootstrap.Channel{chs[0]},
		},
		{
			desc:        "list a subset of existing channels empty",
			domainID:    c.DomainID,
			connections: []string{"5", "6"},
			existing:    []bootstrap.Channel{},
		},
	}
	for _, tc := range cases {
		existing, err := repo.ListExisting(context.Background(), tc.domainID, tc.connections)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.ElementsMatch(t, tc.existing, existing, fmt.Sprintf("%s: Got non-matching elements.", tc.desc))
	}
}

func TestRemoveClient(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	for i := 0; i < 2; i++ {
		err := repo.RemoveClient(context.Background(), saved)
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
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	id := c.Channels[0].ID
	update := bootstrap.Channel{
		ID:       id,
		Name:     "update name",
		Metadata: map[string]any{"update": "metadata update"},
	}
	err = repo.UpdateChannel(context.Background(), update)
	assert.Nil(t, err, fmt.Sprintf("updating config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(context.Background(), c.DomainID, c.ClientID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	var retrieved bootstrap.Channel
	for _, c := range cfg.Channels {
		if c.ID == id {
			retrieved = c
			break
		}
	}
	update.DomainID = retrieved.DomainID
	assert.Equal(t, update, retrieved, fmt.Sprintf("expected %s, go %s", update, retrieved))
}

func TestRemoveChannel(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	c := config
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ClientSecret = uid.String()
	c.ClientID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c, channels)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.RemoveChannel(context.Background(), c.Channels[0].ID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))

	cfg, err := repo.RetrieveByID(context.Background(), c.DomainID, c.ClientID)
	assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.NotContains(t, cfg.Channels, c.Channels[0], fmt.Sprintf("expected to remove channel %s from %s", c.Channels[0], cfg.Channels))
}

func TestConnectClient(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	wrongID := testsutil.GenerateUUID(&testing.T{})

	cases := []struct {
		desc          string
		saveConfig    bool
		initialState  bootstrap.State
		connections   []int
		clientID      string
		channel       int
		err           error
		checkState    bool
		expectedState bootstrap.State
	}{
		{
			desc:          "connect disconnected client",
			saveConfig:    true,
			initialState:  bootstrap.Inactive,
			connections:   []int{0, 1},
			channel:       0,
			err:           nil,
			checkState:    true,
			expectedState: bootstrap.Active,
		},
		{
			desc:          "connect already active client",
			saveConfig:    true,
			initialState:  bootstrap.Active,
			connections:   []int{0, 1},
			channel:       0,
			err:           nil,
			checkState:    true,
			expectedState: bootstrap.Active,
		},
		{
			desc:        "connect non-existent client",
			saveConfig:  true,
			connections: []int{0, 1},
			clientID:    wrongID,
			channel:     0,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:          "connect client without channel connection",
			saveConfig:    true,
			initialState:  bootstrap.Inactive,
			connections:   []int{1},
			channel:       0,
			err:           repoerr.ErrNotFound,
			checkState:    true,
			expectedState: bootstrap.Inactive,
		},
		{
			desc:     "connect empty client",
			clientID: "",
			err:      repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			c := uniqueConfig(t, tc.initialState)
			clientID := tc.clientID
			if tc.saveConfig {
				saved, err := repo.Save(context.Background(), c, selectChannelIDs(c, tc.connections...))
				assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
				if clientID == "" {
					clientID = saved
				}
			}
			channelID := selectChannelID(c, tc.channel)

			err := repo.ConnectClient(context.Background(), channelID, clientID)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: Expected error: %s, got: %s.\n", tc.desc, tc.err, err))

			if tc.checkState {
				cfg, err := repo.RetrieveByID(context.Background(), c.DomainID, c.ClientID)
				assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
				assert.Equal(t, tc.expectedState, cfg.State, fmt.Sprintf("expected %s to have state %d", cfg.ClientID, tc.expectedState))
			}
		})
	}
}

func TestDisconnectClient(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)
	err := deleteChannels(context.Background(), repo)
	require.Nil(t, err, "Channels cleanup expected to succeed.")

	wrongID := testsutil.GenerateUUID(&testing.T{})

	cases := []struct {
		desc          string
		saveConfig    bool
		initialState  bootstrap.State
		connections   []int
		clientID      string
		channel       int
		err           error
		checkState    bool
		expectedState bootstrap.State
	}{
		{
			desc:          "disconnect active connected client",
			saveConfig:    true,
			initialState:  bootstrap.Active,
			connections:   []int{0, 1},
			channel:       0,
			err:           nil,
			checkState:    true,
			expectedState: bootstrap.Inactive,
		},
		{
			desc:          "disconnect already inactive client",
			saveConfig:    true,
			initialState:  bootstrap.Inactive,
			connections:   []int{0, 1},
			channel:       0,
			err:           nil,
			checkState:    true,
			expectedState: bootstrap.Inactive,
		},
		{
			desc:          "disconnect client without channel connection",
			saveConfig:    true,
			initialState:  bootstrap.Active,
			connections:   []int{1},
			channel:       0,
			err:           nil,
			checkState:    true,
			expectedState: bootstrap.Active,
		},
		{
			desc:        "disconnect invalid client",
			saveConfig:  true,
			connections: []int{0, 1},
			clientID:    wrongID,
			channel:     0,
			err:         nil,
		},
		{
			desc:     "disconnect empty client",
			clientID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			c := uniqueConfig(t, tc.initialState)
			clientID := tc.clientID
			if tc.saveConfig {
				saved, err := repo.Save(context.Background(), c, selectChannelIDs(c, tc.connections...))
				assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
				if clientID == "" {
					clientID = saved
				}
			}
			channelID := selectChannelID(c, tc.channel)

			err := repo.DisconnectClient(context.Background(), channelID, clientID)
			assert.Equal(t, tc.err, err, fmt.Sprintf("%s: Expected error: %s, got: %s.\n", tc.desc, tc.err, err))

			if tc.checkState {
				cfg, err := repo.RetrieveByID(context.Background(), c.DomainID, c.ClientID)
				assert.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
				assert.Equal(t, tc.expectedState, cfg.State, fmt.Sprintf("expected %s to have state %d", cfg.ClientID, tc.expectedState))
			}
		})
	}
}

func uniqueConfig(t *testing.T, state bootstrap.State) bootstrap.Config {
	t.Helper()

	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))

	cfg := config
	cfg.ClientSecret = uid.String()
	cfg.ClientID = uid.String()
	cfg.ExternalID = uid.String()
	cfg.ExternalKey = uid.String()
	cfg.State = state
	for i := range cfg.Channels {
		chID, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		cfg.Channels[i].ID = chID.String()
	}
	return cfg
}

func selectChannelIDs(cfg bootstrap.Config, indexes ...int) []string {
	if len(indexes) == 0 {
		ids := make([]string, 0, len(cfg.Channels))
		for _, ch := range cfg.Channels {
			ids = append(ids, ch.ID)
		}
		return ids
	}

	ids := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		ids = append(ids, cfg.Channels[idx].ID)
	}
	return ids
}

func selectChannelID(cfg bootstrap.Config, idx int) string {
	if len(cfg.Channels) == 0 {
		return channels[0]
	}
	return cfg.Channels[idx].ID
}

func deleteChannels(ctx context.Context, repo bootstrap.ConfigRepository) error {
	for _, ch := range channels {
		if err := repo.RemoveChannel(ctx, ch); err != nil {
			return err
		}
	}

	return nil
}
