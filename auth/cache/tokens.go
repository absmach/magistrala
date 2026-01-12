// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

const (
	defKey      = "revoked_tokens"
	defDuration = 15 * time.Minute
)

var _ auth.TokensCache = (*tokensCache)(nil)

type tokensCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewTokensCache returns redis auth cache implementation.
func NewTokensCache(client *redis.Client, duration time.Duration) auth.TokensCache {
	if duration == 0 {
		duration = defDuration
	}
	return &tokensCache{
		client:      client,
		keyDuration: duration,
	}
}

func (tc *tokensCache) Save(ctx context.Context, value string) error {
	if err := tc.client.SAdd(ctx, defKey, value).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (tc *tokensCache) Contains(ctx context.Context, value string) bool {
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
