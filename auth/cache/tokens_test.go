// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/cache"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
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
			desc:    "Save token with long id",
			userID:  userID,
			tokenID: strings.Repeat("a", 513*1024*1024),
			ttl:     10 * time.Minute,
			err:     repoerr.ErrCreateEntity,
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
				ok := tokensCache.IsActive(context.Background(), tc.userID, tc.tokenID)
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
			desc:    "IsActive for different user",
			userID:  testsutil.GenerateUUID(t),
			tokenID: tokenID,
		},
		{
			desc:    "IsActive with long token id",
			userID:  userID,
			tokenID: strings.Repeat("a", 513*1024*1024),
		},
		{
			desc:    "IsActive with empty token id",
			userID:  userID,
			tokenID: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ok := tokensCache.IsActive(context.Background(), tc.userID, tc.tokenID)
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
		{
			desc:    "Remove token with long id from cache",
			userID:  userID,
			tokenID: strings.Repeat("a", 513*1024*1024),
			err:     repoerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.RemoveActive(context.Background(), tc.userID, tc.tokenID)
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				ok := tokensCache.IsActive(context.Background(), tc.userID, tc.tokenID)
				assert.False(t, ok)
			}
		})
	}
}
