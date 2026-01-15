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
	activeTokensKeyPrefix = "active_refresh_tokens:"
	defDuration           = 15 * time.Minute
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

// SaveActive saves an active refresh token ID for a user with TTL.
func (tc *tokensCache) SaveActive(ctx context.Context, userID, tokenID string, ttl time.Duration) error {
	key := activeTokensKeyPrefix + userID

	// Add token ID to the set
	if err := tc.client.SAdd(ctx, key, tokenID).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	// Set expiration for the entire set
	if err := tc.client.Expire(ctx, key, ttl).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

// IsActive checks if the token ID is active for the given user.
func (tc *tokensCache) IsActive(ctx context.Context, userID, tokenID string) bool {
	key := activeTokensKeyPrefix + userID

	ok, err := tc.client.SIsMember(ctx, key, tokenID).Result()
	if err != nil {
		return false
	}

	return ok
}

// RemoveActive removes an active refresh token ID for a user.
func (tc *tokensCache) RemoveActive(ctx context.Context, userID, tokenID string) error {
	key := activeTokensKeyPrefix + userID

	if err := tc.client.SRem(ctx, key, tokenID).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

// RemoveAllActive removes all active refresh tokens for a user.
func (tc *tokensCache) RemoveAllActive(ctx context.Context, userID string) error {
	key := activeTokensKeyPrefix + userID

	if err := tc.client.Del(ctx, key).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}
