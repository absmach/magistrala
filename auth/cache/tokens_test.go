// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/cache"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var (
	storeClient *redis.Client
	storeURL    string
)

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func setupRedisTokensClient() auth.TokensCache {
	return cache.NewTokensCache(storeClient, 10*time.Minute)
}

func TestTokenSave(t *testing.T) {
	storeClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	userID := testsutil.GenerateUUID(t)
	tokenID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc    string
		userID  string
		tokenID string
		ttl     time.Duration
		err     error
	}{
		{
			desc:    "Save active token",
			userID:  userID,
			tokenID: tokenID,
			ttl:     10 * time.Minute,
			err:     nil,
		},
		{
			desc:    "Save already cached token",
			userID:  userID,
			tokenID: tokenID,
			ttl:     10 * time.Minute,
			err:     nil,
		},
		{
			desc:    "Save another token for same user",
			userID:  userID,
			tokenID: testsutil.GenerateUUID(t),
			ttl:     10 * time.Minute,
			err:     nil,
		},
		{
			desc:    "Save token with empty id",
			userID:  userID,
			tokenID: "",
			ttl:     10 * time.Minute,
			err:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.SaveActive(context.Background(), tc.userID, tc.tokenID, tc.ttl)
			if err == nil {
				ok, err := tokensCache.IsActive(context.Background(), tc.tokenID)
				assert.NoError(t, err)
				assert.True(t, ok)
			}
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestTokenContains(t *testing.T) {
	storeClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	userID := testsutil.GenerateUUID(t)
	tokenID := testsutil.GenerateUUID(t)

	err := tokensCache.SaveActive(context.Background(), userID, tokenID, 10*time.Minute)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc    string
		userID  string
		tokenID string
		ok      bool
	}{
		{
			desc:    "IsActive for existing token",
			userID:  userID,
			tokenID: tokenID,
			ok:      true,
		},
		{
			desc:    "IsActive for non existing token",
			userID:  userID,
			tokenID: testsutil.GenerateUUID(t),
		},
		{
			desc:    "IsActive with empty token id",
			userID:  userID,
			tokenID: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ok, err := tokensCache.IsActive(context.Background(), tc.tokenID)
			if tc.ok {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.ok, ok)
		})
	}
}

func TestTokenRemove(t *testing.T) {
	storeClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	userID := testsutil.GenerateUUID(t)
	num := 10
	var tokenIDs []string
	for range num {
		tokenID := testsutil.GenerateUUID(t)
		err := tokensCache.SaveActive(context.Background(), userID, tokenID, 10*time.Minute)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))
		tokenIDs = append(tokenIDs, tokenID)
	}

	cases := []struct {
		desc    string
		userID  string
		tokenID string
		err     error
	}{
		{
			desc:    "Remove an existing token from cache",
			userID:  userID,
			tokenID: tokenIDs[0],
			err:     nil,
		},
		{
			desc:    "Remove token with empty id from cache",
			userID:  userID,
			tokenID: "",
			err:     nil,
		},
		{
			desc:    "Remove non existing id from cache",
			userID:  userID,
			tokenID: testsutil.GenerateUUID(t),
			err:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.RemoveActive(context.Background(), tc.tokenID)
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				ok, err := tokensCache.IsActive(context.Background(), tc.tokenID)
				assert.NoError(t, err)
				assert.False(t, ok)
			}
		})
	}
}

func TestListUserTokens(t *testing.T) {
	storeClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	userID := testsutil.GenerateUUID(t)
	userID2 := testsutil.GenerateUUID(t)
	num := 5
	var tokenIDs []string

	for range num {
		tokenID := testsutil.GenerateUUID(t)
		err := tokensCache.SaveActive(context.Background(), userID, tokenID, 10*time.Minute)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))
		tokenIDs = append(tokenIDs, tokenID)
	}

	tokenID2 := testsutil.GenerateUUID(t)
	err := tokensCache.SaveActive(context.Background(), userID2, tokenID2, 10*time.Minute)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc           string
		userID         string
		expectedCount  int
		expectedTokens []string
		err            error
	}{
		{
			desc:           "List all tokens for user with multiple tokens",
			userID:         userID,
			expectedCount:  num,
			expectedTokens: tokenIDs,
			err:            nil,
		},
		{
			desc:           "List tokens for user with single token",
			userID:         userID2,
			expectedCount:  1,
			expectedTokens: []string{tokenID2},
			err:            nil,
		},
		{
			desc:           "List tokens for user with no tokens",
			userID:         testsutil.GenerateUUID(t),
			expectedCount:  0,
			expectedTokens: nil,
			err:            nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tokens, err := tokensCache.ListUserTokens(context.Background(), tc.userID)
			assert.True(t, errors.Contains(err, tc.err))
			assert.Equal(t, tc.expectedCount, len(tokens))
			if tc.expectedTokens != nil {
				assert.ElementsMatch(t, tc.expectedTokens, tokens)
			}
		})
	}

	t.Run("Cleanup expired tokens from list", func(t *testing.T) {
		// Remove one token directly from Redis to simulate expiration
		err := tokensCache.RemoveActive(context.Background(), tokenIDs[0])
		assert.NoError(t, err)

		// List should now return only valid tokens
		tokens, err := tokensCache.ListUserTokens(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, num-1, len(tokens))
		assert.NotContains(t, tokens, tokenIDs[0])
	})
}
