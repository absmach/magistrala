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
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/absmach/magistrala/pkg/testsutil"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const numConfigs = 10

var config = bootstrap.Config{
	ID:          "mg-client",
	ExternalID:  "external-id",
	ExternalKey: "external-key",
	WorkspaceID: testsutil.GenerateUUID(&testing.T{}),
	Content:     "content",
	Status:      bootstrap.DisabledStatus,
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
		desc        string
		workspaceID string
		id          string
		err         error
	}{
		{
			desc:        "retrieve config",
			workspaceID: c.WorkspaceID,
			id:          id,
			err:         nil,
		},
		{
			desc:        "retrieve config with wrong domain ID ",
			workspaceID: "2",
			id:          id,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:        "retrieve a non-existing config",
			workspaceID: c.WorkspaceID,
			id:          nonexistentConfID.String(),
			err:         repoerr.ErrNotFound,
		},
		{
			desc:        "retrieve a config with invalid ID",
			workspaceID: c.WorkspaceID,
			id:          "invalid",
			err:         repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases {
		_, err := repo.RetrieveByID(context.Background(), tc.workspaceID, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveAll(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	for i := 0; i < numConfigs; i++ {
		c := config

		// Use UUID to prevent conflict errors.
		uid, err := uuid.NewV4()
		require.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
		c.ExternalID = uid.String()
		c.Name = fmt.Sprintf("name %d", i)
		c.ID = uid.String()

		if i%2 == 0 {
			c.Status = bootstrap.EnabledStatus
		}

		_, err = repo.Save(context.Background(), c)
		require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	}
	cases := []struct {
		desc        string
		workspaceID string
		offset      uint64
		limit       uint64
		filter      bootstrap.Filter
		size        int
	}{
		{
			desc:        "retrieve all configs",
			workspaceID: config.WorkspaceID,
			offset:      0,
			limit:       uint64(numConfigs),
			size:        numConfigs,
		},
		{
			desc:        "retrieve a subset of configs",
			workspaceID: config.WorkspaceID,
			offset:      5,
			limit:       uint64(numConfigs - 5),
			size:        numConfigs - 5,
		},
		{
			desc:        "retrieve with wrong domain ID ",
			workspaceID: "2",
			offset:      0,
			limit:       uint64(numConfigs),
			size:        0,
		},
		{
			desc:        "retrieve all active configs ",
			workspaceID: config.WorkspaceID,
			offset:      0,
			limit:       uint64(numConfigs),
			filter:      bootstrap.Filter{FullMatch: map[string]string{"status": bootstrap.EnabledStatus.String()}},
			size:        numConfigs / 2,
		},
		{
			desc:        "retrieve all with partial match filter",
			workspaceID: config.WorkspaceID,
			offset:      0,
			limit:       uint64(numConfigs),
			filter:      bootstrap.Filter{PartialMatch: map[string]string{"name": "1"}},
			size:        1,
		},
		{
			desc:        "retrieve search by name",
			workspaceID: config.WorkspaceID,
			offset:      0,
			limit:       uint64(numConfigs),
			filter:      bootstrap.Filter{PartialMatch: map[string]string{"name": "1"}},
			size:        1,
		},
	}
	for _, tc := range cases {
		ret, err := repo.RetrieveAll(context.Background(), tc.workspaceID, tc.filter, tc.offset, tc.limit)
		assert.NoError(t, err, fmt.Sprintf("%s: unexpected error %v\n", tc.desc, err))
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

	withRenderContext := c
	withRenderContext.RenderContext = map[string]any{
		"site":   "warehouse-2",
		"region": "mombasa",
	}

	wrongWorkspaceID := c
	wrongWorkspaceID.WorkspaceID = "3"

	cases := []struct {
		desc          string
		config        bootstrap.Config
		renderContext map[string]any
		err           error
	}{
		{
			desc:   "update with wrong workspaceID",
			config: wrongWorkspaceID,
			err:    repoerr.ErrNotFound,
		},
		{
			desc:   "update a config",
			config: c,
			err:    nil,
		},
		{
			desc:          "update a config render_context",
			config:        withRenderContext,
			renderContext: map[string]any{"site": "warehouse-2", "region": "mombasa"},
			err:           nil,
		},
	}
	for _, tc := range cases {
		err := repo.Update(context.Background(), tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil && tc.renderContext != nil {
			saved, err := repo.RetrieveByID(context.Background(), tc.config.WorkspaceID, tc.config.ID)
			require.Nil(t, err, fmt.Sprintf("%s: unexpected retrieve error: %s\n", tc.desc, err))
			assert.Equal(t, tc.renderContext, saved.RenderContext, fmt.Sprintf("%s: expected render_context %v got %v\n", tc.desc, tc.renderContext, saved.RenderContext))
		}
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

	wrongWorkspaceID := c
	wrongWorkspaceID.WorkspaceID = "3"

	cases := []struct {
		desc           string
		configID       string
		workspaceID    string
		cert           string
		certKey        string
		ca             string
		expectedConfig bootstrap.Config
		err            error
	}{
		{
			desc:           "update with wrong domain ID ",
			configID:       "",
			cert:           "cert",
			certKey:        "certKey",
			ca:             "",
			workspaceID:    wrongWorkspaceID.WorkspaceID,
			expectedConfig: bootstrap.Config{},
			err:            repoerr.ErrNotFound,
		},
		{
			desc:        "update a config",
			configID:    c.ID,
			cert:        "cert",
			certKey:     "certKey",
			ca:          "ca",
			workspaceID: c.WorkspaceID,
			expectedConfig: bootstrap.Config{
				ID:          c.ID,
				ClientCert:  "cert",
				CACert:      "ca",
				ClientKey:   "certKey",
				WorkspaceID: c.WorkspaceID,
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		cfg, err := repo.UpdateCert(context.Background(), tc.workspaceID, tc.configID, tc.cert, tc.certKey, tc.ca)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.expectedConfig, cfg, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.expectedConfig, cfg))
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

	err = repo.Remove(context.Background(), c.WorkspaceID, id)
	assert.Nil(t, err, fmt.Sprintf("failed to remove config due to: %s", err))

	_, err = repo.RetrieveByID(context.Background(), c.WorkspaceID, id)
	assert.True(t, errors.Contains(err, repoerr.ErrNotFound), fmt.Sprintf("expected %s got %s", repoerr.ErrNotFound, err))

	// Removing a config that is not in this workspace must report
	// ErrNotFound rather than succeeding: the caller would otherwise be told
	// the config was deleted, and atomService.Remove would tear down the
	// Atom projection of a config that still exists.
	err = repo.Remove(context.Background(), c.WorkspaceID, id)
	assert.True(t, errors.Contains(err, repoerr.ErrNotFound), fmt.Sprintf("expected %s got %s", repoerr.ErrNotFound, err))
}

func TestRemoveOtherWorkspace(t *testing.T) {
	repo := postgres.NewConfigRepository(db, testLog)

	c := config
	uid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("Got unexpected error: %s.\n", err))
	c.ID = uid.String()
	c.ExternalID = uid.String()
	c.ExternalKey = uid.String()
	id, err := repo.Save(context.Background(), c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	err = repo.Remove(context.Background(), "another-workspace", id)
	assert.True(t, errors.Contains(err, repoerr.ErrNotFound), fmt.Sprintf("expected %s got %s", repoerr.ErrNotFound, err))

	// The config must survive a delete addressed to the wrong workspace.
	_, err = repo.RetrieveByID(context.Background(), c.WorkspaceID, id)
	assert.Nil(t, err, fmt.Sprintf("config should still exist: %s", err))
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
		desc        string
		workspaceID string
		id          string
		status      bootstrap.Status
		err         error
	}{
		{
			desc:        "change status with wrong domain ID ",
			id:          saved,
			workspaceID: "2",
			err:         repoerr.ErrNotFound,
		},
		{
			desc:        "change status with wrong id",
			id:          "wrong",
			workspaceID: c.WorkspaceID,
			err:         repoerr.ErrNotFound,
		},
		{
			desc:        "change status to Active",
			id:          saved,
			workspaceID: c.WorkspaceID,
			status:      bootstrap.EnabledStatus,
			err:         nil,
		},
		{
			desc:        "change status to Inactive",
			id:          saved,
			workspaceID: c.WorkspaceID,
			status:      bootstrap.DisabledStatus,
			err:         nil,
		},
	}
	for _, tc := range cases {
		err := repo.ChangeStatus(context.Background(), tc.workspaceID, tc.id, tc.status)
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
		ID:            profileID,
		WorkspaceID:   c.WorkspaceID,
		Name:          "edge-gateway",
		ContentFormat: bootstrap.ContentFormatGoTemplate,
		Version:       1,
	})
	require.Nil(t, err, fmt.Sprintf("Saving profile expected to succeed: %s.\n", err))

	err = configRepo.AssignProfile(context.Background(), c.WorkspaceID, saved, profileID)
	require.Nil(t, err, fmt.Sprintf("Assigning profile expected to succeed: %s.\n", err))

	stored, err := configRepo.RetrieveByID(context.Background(), c.WorkspaceID, saved)
	require.Nil(t, err, fmt.Sprintf("Retrieving config expected to succeed: %s.\n", err))
	assert.Equal(t, profileID, stored.ProfileID, "expected profile assignment to round-trip through the repository")
}
