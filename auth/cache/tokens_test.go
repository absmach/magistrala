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

func setupRedisTokensClient() auth.UserActiveTokensCache {
	return cache.NewUserActiveTokensCache(storeClient, 10*time.Minute)
}

func TestTokenSave(t *testing.T) {
	storeClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	userID := testsutil.GenerateUUID(t)
	tokenID := testsutil.GenerateUUID(t)

	cases := []struct {
		desc        string
		userID      string
		tokenID     string
		description string
		ttl         time.Duration
		err         error
	}{
		{
			desc:        "Save active token",
			userID:      userID,
			tokenID:     tokenID,
			description: "Test token",
			ttl:         10 * time.Minute,
			err:         nil,
		},
		{
			desc:        "Save already cached token",
			userID:      userID,
			tokenID:     tokenID,
			description: "Updated token",
			ttl:         10 * time.Minute,
			err:         nil,
		},
		{
			desc:        "Save another token for same user",
			userID:      userID,
			tokenID:     testsutil.GenerateUUID(t),
			description: "Another token",
			ttl:         10 * time.Minute,
			err:         nil,
		},
		{
			desc:        "Save token with empty id",
			userID:      userID,
			tokenID:     "",
			description: "Empty ID token",
			ttl:         10 * time.Minute,
			err:         nil,
		},
		{
			desc:        "Save token with empty description",
			userID:      userID,
			tokenID:     testsutil.GenerateUUID(t),
			description: "",
			ttl:         10 * time.Minute,
			err:         nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.SaveActive(context.Background(), tc.userID, tc.tokenID, tc.description, tc.ttl)
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

	err := tokensCache.SaveActive(context.Background(), userID, tokenID, "Test token", 10*time.Minute)
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
	for i := range num {
		tokenID := testsutil.GenerateUUID(t)
		err := tokensCache.SaveActive(context.Background(), userID, tokenID, fmt.Sprintf("Token %d", i), 10*time.Minute)
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
	var expectedTokens []auth.TokenInfo

	for i := range num {
		tokenID := testsutil.GenerateUUID(t)
		description := fmt.Sprintf("Token %d", i)
		err := tokensCache.SaveActive(context.Background(), userID, tokenID, description, 10*time.Minute)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))
		expectedTokens = append(expectedTokens, auth.TokenInfo{
			ID:          tokenID,
			Description: description,
		})
	}

	tokenID2 := testsutil.GenerateUUID(t)
	desc2 := "User 2 token"
	err := tokensCache.SaveActive(context.Background(), userID2, tokenID2, desc2, 10*time.Minute)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc           string
		userID         string
		expectedCount  int
		expectedTokens []auth.TokenInfo
		err            error
	}{
		{
			desc:           "List all tokens for user with multiple tokens",
			userID:         userID,
			expectedCount:  num,
			expectedTokens: expectedTokens,
			err:            nil,
		},
		{
			desc:           "List tokens for user with single token",
			userID:         userID2,
			expectedCount:  1,
			expectedTokens: []auth.TokenInfo{{ID: tokenID2, Description: desc2}},
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
		err := tokensCache.RemoveActive(context.Background(), expectedTokens[0].ID)
		assert.NoError(t, err)

		// List should now return only valid tokens
		tokens, err := tokensCache.ListUserTokens(context.Background(), userID)
		assert.NoError(t, err)
		assert.Equal(t, num-1, len(tokens))

		// Check that the removed token is not in the list
		for _, token := range tokens {
			assert.NotEqual(t, expectedTokens[0].ID, token.ID)
		}
	})
}
