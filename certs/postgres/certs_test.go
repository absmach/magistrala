// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"testing"

	"github.com/absmach/supermq/certs/postgres"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	entityID     = "bfead30d-5a1d-40f3-be21-fd8ffad49db0"
	serialNumber = "20:f4:bd:43:2c:c7:06:82:c7:f2:00:47:51:b6:81:6f:fa:c4:46:0c"
)

func TestSaveCertEntityMapping(t *testing.T) {
	repo := postgres.NewRepository(database)

	testCases := []struct {
		desc         string
		serialNumber string
		entityID     string
		err          error
	}{
		{
			desc:         "successful save",
			serialNumber: serialNumber,
			entityID:     entityID,
			err:          nil,
		},
		{
			desc:         "save duplicate mapping",
			serialNumber: serialNumber,
			entityID:     entityID,
			err:          postgres.ErrConflict,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.SaveCertEntityMapping(context.Background(), tc.serialNumber, tc.entityID)
			if tc.err != nil {
				require.Error(t, err)
				assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetEntityIDBySerial(t *testing.T) {
	repo := postgres.NewRepository(database)

	// Setup: save a mapping first
	testSerial := "test-serial-456"
	testEntityID := "test-entity-789"
	err := repo.SaveCertEntityMapping(context.Background(), testSerial, testEntityID)
	require.NoError(t, err)

	testCases := []struct {
		desc         string
		serialNumber string
		expectedID   string
		err          error
	}{
		{
			desc:         "successful retrieval",
			serialNumber: testSerial,
			expectedID:   testEntityID,
			err:          nil,
		},
		{
			desc:         "serial number not found",
			serialNumber: "non-existent-serial",
			expectedID:   "",
			err:          postgres.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			entityID, err := repo.GetEntityIDBySerial(context.Background(), tc.serialNumber)
			if tc.err != nil {
				require.Error(t, err)
				assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
				assert.Empty(t, entityID)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedID, entityID)
			}
		})
	}
}

func TestListCertsByEntityID(t *testing.T) {
	repo := postgres.NewRepository(database)

	// Setup: save multiple mappings for the same entity
	testEntityID := "test-entity-list"
	testSerials := []string{"serial-1", "serial-2", "serial-3"}

	for _, serial := range testSerials {
		err := repo.SaveCertEntityMapping(context.Background(), serial, testEntityID)
		require.NoError(t, err)
	}

	testCases := []struct {
		desc             string
		entityID         string
		expectedCount    int
		expectedContains []string
		err              error
	}{
		{
			desc:             "successful list with multiple certs",
			entityID:         testEntityID,
			expectedCount:    3,
			expectedContains: testSerials,
			err:              nil,
		},
		{
			desc:             "entity with no certificates",
			entityID:         "non-existent-entity",
			expectedCount:    0,
			expectedContains: []string{},
			err:              nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			serials, err := repo.ListCertsByEntityID(context.Background(), tc.entityID)
			if tc.err != nil {
				require.Error(t, err)
				assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, serials, tc.expectedCount)

				if tc.expectedCount > 0 {
					for _, expectedSerial := range tc.expectedContains {
						assert.Contains(t, serials, expectedSerial)
					}
				}
			}
		})
	}
}

func TestRemoveCertEntityMapping(t *testing.T) {
	repo := postgres.NewRepository(database)

	testSerial := "test-serial-remove"
	testEntityID := "test-entity-remove"
	err := repo.SaveCertEntityMapping(context.Background(), testSerial, testEntityID)
	require.NoError(t, err)

	testCases := []struct {
		desc         string
		serialNumber string
		err          error
	}{
		{
			desc:         "successful removal",
			serialNumber: testSerial,
			err:          nil,
		},
		{
			desc:         "remove non-existent mapping",
			serialNumber: "non-existent-serial",
			err:          postgres.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := repo.RemoveCertEntityMapping(context.Background(), tc.serialNumber)
			if tc.err != nil {
				require.Error(t, err)
				assert.True(t, errors.Contains(err, tc.err), "expected error %v, got %v", tc.err, err)
			} else {
				require.NoError(t, err)

				// Verify the mapping was actually removed
				_, err := repo.GetEntityIDBySerial(context.Background(), tc.serialNumber)
				assert.True(t, errors.Contains(err, postgres.ErrNotFound))
			}
		})
	}
}

func TestCertEntityMappingWorkflow(t *testing.T) {
	repo := postgres.NewRepository(database)

	// Test complete workflow: save -> get -> list -> remove
	entityID := "workflow-entity"
	serials := []string{"workflow-serial-1", "workflow-serial-2"}

	// Save mappings
	for _, serial := range serials {
		err := repo.SaveCertEntityMapping(context.Background(), serial, entityID)
		require.NoError(t, err)
	}

	// Verify we can get entity IDs by serial
	for _, serial := range serials {
		retrievedID, err := repo.GetEntityIDBySerial(context.Background(), serial)
		require.NoError(t, err)
		assert.Equal(t, entityID, retrievedID)
	}

	// Verify we can list all serials for the entity
	listedSerials, err := repo.ListCertsByEntityID(context.Background(), entityID)
	require.NoError(t, err)
	assert.Len(t, listedSerials, 2)
	for _, serial := range serials {
		assert.Contains(t, listedSerials, serial)
	}

	// Remove one mapping
	err = repo.RemoveCertEntityMapping(context.Background(), serials[0])
	require.NoError(t, err)

	// Verify it's removed
	_, err = repo.GetEntityIDBySerial(context.Background(), serials[0])
	assert.True(t, errors.Contains(err, postgres.ErrNotFound))

	// Verify the other mapping still exists
	retrievedID, err := repo.GetEntityIDBySerial(context.Background(), serials[1])
	require.NoError(t, err)
	assert.Equal(t, entityID, retrievedID)

	// Verify list now shows only one serial
	listedSerials, err = repo.ListCertsByEntityID(context.Background(), entityID)
	require.NoError(t, err)
	assert.Len(t, listedSerials, 1)
	assert.Contains(t, listedSerials, serials[1])
}
