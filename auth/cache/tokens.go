// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/redis/go-redis/v9"
)

const (
	defDuration   = 15 * time.Minute
	refreshPrefix = "refresh_tokens:"
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
	pipe := tc.client.TxPipeline()

	pipe.Set(ctx, tc.tokenKey(tokenID), userID, ttl)
	pipe.SAdd(ctx, tc.userTokensKey(userID), tokenID)

	_, err := pipe.Exec(ctx)

	return err
}

// IsActive checks if the token ID is active for the given user.
func (tc *tokensCache) IsActive(ctx context.Context, tokenID string) (bool, error) {
	count, err := tc.client.Exists(ctx, tc.tokenKey(tokenID)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListUserTokens lists all active refresh token IDs for a user.
func (tc *tokensCache) ListUserTokens(ctx context.Context, userID string) ([]string, error) {
	key := tc.userTokensKey(userID)
	tokenIDs, err := tc.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(tokenIDs) == 0 {
		return nil, nil
	}

	valid := make([]string, 0, len(tokenIDs))
	pipe := tc.client.Pipeline()

	existsCmds := make(map[string]*redis.IntCmd, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		existsCmds[tokenID] = pipe.Exists(ctx, tc.tokenKey(tokenID))
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	cleanup := tc.client.Pipeline()
	for tokenID, cmd := range existsCmds {
		switch {
		case cmd.Val() == 1:
			valid = append(valid, tokenID)
		default:
			cleanup.SRem(ctx, key, tokenID)
		}
	}

	_, err = cleanup.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return valid, nil
}

// RemoveActive removes an active refresh token ID for a user.
func (tc *tokensCache) RemoveActive(ctx context.Context, tokenID string) error {
	tokenKey := tc.tokenKey(tokenID)

	userID, err := tc.client.Get(ctx, tokenKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	pipe := tc.client.TxPipeline()
	pipe.Del(ctx, tokenKey)
	pipe.SRem(ctx, tc.userTokensKey(userID), tokenID)

	_, err = pipe.Exec(ctx)
	return err
}

func (tc *tokensCache) tokenKey(tokenID string) string {
	return fmt.Sprintf("%s:token:%s", refreshPrefix, tokenID)
}

func (tc *tokensCache) userTokensKey(userID string) string {
	return fmt.Sprintf("%s:user_tokens:%s", refreshPrefix, userID)
}
