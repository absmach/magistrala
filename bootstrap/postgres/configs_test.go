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

var config = bootstrap.Config{
	ID:          "mg-client",
	ExternalID:  "external-id",
	ExternalKey: "external-key",
	DomainID:    testsutil.GenerateUUID(&testing.T{}),
	Content:     "content",
	Status:      bootstrap.Inactive,
}

func TestSave(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	diff := "different"

	duplicateClient := config
	duplicateClient.ExternalID = diff

	duplicateExternal := config
	duplicateExternal.ID = diff

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
			desc:   "save config with same Client ID",
			config: duplicateClient,
			err:    repoerr.ErrConflict,
		},
		{
			desc:   "save config with same external ID",
			config: duplicateExternal,
			err:    repoerr.ErrConflict,
		},
	}
	for _, tc := range cases {
		id, err := repo.Save(context.Background(), tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, id, tc.config.ID, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.config.ID, id))
		}
	}
}

func TestRetrieveByID(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c)
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

	clientIDs := make([]string, numConfigs)

	for i := 0; i < numConfigs; i++ {
		c := config

		// Use UUID to prevent conflict errors.
		uid, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = uid.String()
		c.Name = fmt.Sprintf("name %d", i)
		c.ID = uid.String()

		clientIDs[i] = c.ID

		if i%2 == 0 {
			c.Status = bootstrap.Active
		}

		_, err = repo.Save(context.Background(), c)
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
			filter:   bootstrap.Filter{FullMatch: map[string]string{"status": bootstrap.Active.String()}},
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

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c)
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

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c)
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

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	_, err = repo.Save(context.Background(), c)
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
			clientID: c.ID,
			cert:     "cert",
			certKey:  "certKey",
			ca:       "ca",
			domainID: c.DomainID,
			expectedConfig: bootstrap.Config{
				ID:         c.ID,
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

func TestRemove(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c)
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

func TestChangeStatus(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	cases := []struct {
		desc     string
		domainID string
		id       string
		status   bootstrap.Status
		err      error
	}{
		{
			desc:     "change status with wrong domain ID ",
			id:       saved,
			domainID: "2",
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "change status with wrong id",
			id:       "wrong",
			domainID: c.DomainID,
			err:      repoerr.ErrNotFound,
		},
		{
			desc:     "change status to Active",
			id:       saved,
			domainID: c.DomainID,
			status:   bootstrap.Active,
			err:      nil,
		},
		{
			desc:     "change status to Inactive",
			id:       saved,
			domainID: c.DomainID,
			status:   bootstrap.Inactive,
			err:      nil,
		},
	}
	for _, tc := range cases {
		err := repo.ChangeStatus(context.Background(), tc.domainID, tc.id, tc.status)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAssignProfile(t *testing.T) {
	configRepo := postgres.NewConfigRepository(db, testLog)
	profileRepo := postgres.NewProfileRepository(db, testLog)

	c := config
	uid, err := uuid.NewV4()
	require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := configRepo.Save(context.Background(), c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	profileID := testsutil.GenerateUUID(t)
	_, err = profileRepo.Save(context.Background(), bootstrap.Profile{
		ID:             profileID,
		DomainID:       c.DomainID,
		Name:           "edge-gateway",
		TemplateFormat: bootstrap.TemplateFormatGoTemplate,
		Version:        1,
	})
	require.Nil(t, err, fmt.Sprintf("Saving profile expected to succeed: %s.\n", err))

	err = configRepo.AssignProfile(context.Background(), c.DomainID, saved, profileID)
	require.Nil(t, err, fmt.Sprintf("Assigning profile expected to succeed: %s.\n", err))

	stored, err := configRepo.RetrieveByID(context.Background(), c.DomainID, saved)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.Equal(t, profileID, stored.ProfileID, "expected profile assignment to round-trip through the repository")
}

func TestRemoveClient(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	c := config
	// Use UUID to prevent conflicts.
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	saved, err := repo.Save(context.Background(), c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	for i := 0; i < 2; i++ {
		err := repo.RemoveClient(context.Background(), saved)
		assert.Nil(t, err, fmt.Sprintf("an unexpected error occurred: %s\n", err))
	}
}
