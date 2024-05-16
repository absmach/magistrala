// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

const defKey = "revoked_tokens"

var _ auth.Cache = (*tokensCache)(nil)

type tokensCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewTokensCache returns redis auth cache implementation.
func NewTokensCache(client *redis.Client, duration time.Duration) auth.Cache {
	return &tokensCache{
		client:      client,
		keyDuration: duration,
	}
}

func (tc *tokensCache) Save(ctx context.Context, _, value string) error {
	if err := tc.client.SAdd(ctx, defKey, value, tc.keyDuration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (tc *tokensCache) Contains(ctx context.Context, _, value string) bool {
	ok, err := tc.client.SIsMember(ctx, defKey, value).Result()
	if err != nil {
		return false
	}

	return ok
}

func (tc *tokensCache) Remove(ctx context.Context, value string) error {
	if err := tc.client.SRem(ctx, defKey, value).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}
