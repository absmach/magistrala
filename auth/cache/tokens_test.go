// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/cache"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/stretchr/testify/assert"
)

func setupRedisTokensClient() auth.TokensCache {
	return cache.NewTokensCache(redisClient, 10*time.Minute)
}

func TestTokenSave(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	key := auth.Key{
		ID: testsutil.GenerateUUID(t),
	}
	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "Save token",
			key:  key,
			err:  nil,
		},
		{
			desc: "Save already cached policy",
			key:  key,
			err:  nil,
		},
		{
			desc: "Save another policy",
			key: auth.Key{
				ID: testsutil.GenerateUUID(t),
			},
			err: nil,
		},
		{
			desc: "Save policy with long key",
			key: auth.Key{
				ID: strings.Repeat("a", 513*1024*1024),
			},
			err: repoerr.ErrCreateEntity,
		},
		{
			desc: "Save policy with empty key",
			err:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.Save(context.Background(), "", tc.key.ID)
			if err == nil {
				ok := tokensCache.Contains(context.Background(), "", tc.key.ID)
				assert.True(t, ok)
			}
			assert.True(t, errors.Contains(err, tc.err))
		})
	}
}

func TestTokenContains(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	key := auth.Key{
		ID: testsutil.GenerateUUID(t),
	}

	err := tokensCache.Save(context.Background(), "", key.ID)
	assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))

	cases := []struct {
		desc string
		key  auth.Key
		ok   bool
	}{
		{
			desc: "Contains existing key",
			key:  key,
			ok:   true,
		},
		{
			desc: "Contains non existing key",
			key: auth.Key{
				ID: testsutil.GenerateUUID(t),
			},
		},
		{
			desc: "Contains key with long id",
			key: auth.Key{
				ID: strings.Repeat("a", 513*1024*1024),
			},
		},
		{
			desc: "Contains key with empty id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ok := tokensCache.Contains(context.Background(), "", tc.key.ID)
			assert.Equal(t, tc.ok, ok)
		})
	}
}

func TestTokenRemove(t *testing.T) {
	redisClient.FlushAll(context.Background())
	tokensCache := setupRedisTokensClient()

	num := 10
	var ids []string
	for range num {
		id := testsutil.GenerateUUID(t)
		err := tokensCache.Save(context.Background(), "", id)
		assert.Nil(t, err, fmt.Sprintf("Unexpected error while trying to save: %s", err))
		ids = append(ids, id)
	}

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "Remove an existing token from cache",
			id:   ids[0],
			err:  nil,
		},
		{
			desc: "Remove token with empty id from cache",
			err:  nil,
		},
		{
			desc: "Remove non existing id from cache",
			id:   testsutil.GenerateUUID(t),
			err:  nil,
		},
		{
			desc: "Remove token with long id from cache",
			id:   strings.Repeat("a", 513*1024*1024),
			err:  repoerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.Remove(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err))
			if err == nil {
				ok := tokensCache.Contains(context.Background(), "", tc.id)
				assert.False(t, ok)
			}
		})
	}
}
