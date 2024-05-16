// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/cache"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

var key = auth.Key{
	ID: testsutil.GenerateUUID(&testing.T{}),
}

func setupRedisTokensClient(t *testing.T) auth.Cache {
	opts, err := redis.ParseURL(redisURL)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error on parsing redis URL: %s", err))
	redisClient := redis.NewClient(opts)
	return cache.NewPoliciesCache(redisClient, 10*time.Minute)
}

func TestTokenSave(t *testing.T) {
	tokensCache := setupRedisTokensClient(t)

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
				ID: testsutil.GenerateUUID(&testing.T{}),
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
	tokensCache := setupRedisTokensClient(t)

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
				ID: testsutil.GenerateUUID(&testing.T{}),
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
	tokensCache := setupRedisTokensClient(t)

	num := 1000
	var ids []string
	for i := 0; i < num; i++ {
		id := testsutil.GenerateUUID(&testing.T{})
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
			desc: "Remove an existing id from cache",
			id:   ids[0],
			err:  nil,
		},
		{
			desc: "Remove multiple existing id from cache",
			id:   "*",
			err:  nil,
		},
		{
			desc: "Remove non existing id from cache",
			id:   testsutil.GenerateUUID(&testing.T{}),
			err:  nil,
		},
		{
			desc: "Remove policy with empty id from cache",
			err:  nil,
		},
		{
			desc: "Remove policy with long id from cache",
			id:   strings.Repeat("a", 513*1024*1024),
			err:  repoerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tokensCache.Remove(context.Background(), tc.id)
			assert.True(t, errors.Contains(err, tc.err))
			if tc.id == "*" {
				for _, id := range ids {
					ok := tokensCache.Contains(context.Background(), "", id)
					assert.False(t, ok)
				}
				return
			}
			if err == nil {
				ok := tokensCache.Contains(context.Background(), "", tc.id)
				assert.False(t, ok)
			}
		})
	}
}
