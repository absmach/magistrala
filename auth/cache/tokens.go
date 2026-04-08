// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"strconv"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/redis/go-redis/v9"
)

const (
	refreshPrefix = "refresh_tokens:"
	scoreNegInf   = "-inf"
	scorePosInf   = "+inf"
)

var _ auth.UserActiveTokensCache = (*tokensCache)(nil)

type tokensCache struct {
	client *redis.Client
}

// NewUserActiveTokensCache returns redis auth cache implementation.
func NewUserActiveTokensCache(client *redis.Client) (auth.UserActiveTokensCache, error) {
	return &tokensCache{client: client}, nil
}

// SaveActive saves an active refresh token ID for a user with optional description.
func (tc *tokensCache) SaveActive(ctx context.Context, userID, tokenID, description string, expiry time.Time) error {
	ttl := time.Until(expiry)

	pipe := tc.client.TxPipeline()

	pipe.Set(ctx, tokenKey(tokenID), description, ttl)
	pipe.ZAdd(ctx, userTokensKey(userID), redis.Z{
		Score:  float64(expiry.Unix()),
		Member: tokenID,
	})

	_, err := pipe.Exec(ctx)

	return err
}

// IsActive checks if the token ID is active for the given user.
func (tc *tokensCache) IsActive(ctx context.Context, tokenID string) (bool, error) {
	count, err := tc.client.Exists(ctx, tokenKey(tokenID)).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListUserTokens lists all active refresh token IDs with descriptions for a user.
func (tc *tokensCache) ListUserTokens(ctx context.Context, userID string) ([]auth.TokenInfo, error) {
	key := userTokensKey(userID)
	now := strconv.FormatInt(time.Now().Unix(), 10)

	pipe := tc.client.TxPipeline()
	pipe.ZRemRangeByScore(ctx, key, scoreNegInf, now)
	zrangeCmd := pipe.ZRangeByScore(ctx, key, &redis.ZRangeBy{Min: "(" + now, Max: scorePosInf})
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	tokenIDs, err := zrangeCmd.Result()
	if err != nil {
		return nil, err
	}

	if len(tokenIDs) == 0 {
		return nil, nil
	}

	getPipe := tc.client.Pipeline()
	getCmds := make([]*redis.StringCmd, len(tokenIDs))
	for i, tokenID := range tokenIDs {
		getCmds[i] = getPipe.Get(ctx, tokenKey(tokenID))
	}

	if _, err = getPipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	valid := make([]auth.TokenInfo, 0, len(tokenIDs))
	for i, cmd := range getCmds {
		description, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}

		valid = append(valid, auth.TokenInfo{
			ID:          tokenIDs[i],
			Description: description,
		})
	}

	return valid, nil
}

// RemoveActive removes an active refresh token ID for a user.
func (tc *tokensCache) RemoveActive(ctx context.Context, userID, tokenID string) error {
	pipe := tc.client.TxPipeline()
	pipe.Del(ctx, tokenKey(tokenID))
	pipe.ZRem(ctx, userTokensKey(userID), tokenID)

	_, err := pipe.Exec(ctx)
	return err
}

func tokenKey(tokenID string) string {
	return refreshPrefix + "token:" + tokenID
}

func userTokensKey(userID string) string {
	return refreshPrefix + "user_tokens:" + userID
}
