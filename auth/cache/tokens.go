// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/redis/go-redis/v9"
)

const (
	defDuration   = 15 * time.Minute
	refreshPrefix = "refresh_tokens:"
)

type tokenData struct {
	UserID      string `json:"user_id"`
	Description string `json:"description,omitempty"`
}

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

// SaveActive saves an active refresh token ID for a user with TTL and optional description.
func (tc *tokensCache) SaveActive(ctx context.Context, userID, tokenID, description string, ttl time.Duration) error {
	data := tokenData{
		UserID:      userID,
		Description: description,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	pipe := tc.client.TxPipeline()

	pipe.Set(ctx, tc.tokenKey(tokenID), dataJSON, ttl)
	pipe.SAdd(ctx, tc.userTokensKey(userID), tokenID)

	_, err = pipe.Exec(ctx)

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

// ListUserTokens lists all active refresh token IDs with descriptions for a user.
func (tc *tokensCache) ListUserTokens(ctx context.Context, userID string) ([]auth.TokenInfo, error) {
	key := tc.userTokensKey(userID)
	tokenIDs, err := tc.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	if len(tokenIDs) == 0 {
		return nil, nil
	}

	valid := make([]auth.TokenInfo, 0, len(tokenIDs))
	pipe := tc.client.Pipeline()

	getCmds := make(map[string]*redis.StringCmd, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		getCmds[tokenID] = pipe.Get(ctx, tc.tokenKey(tokenID))
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	cleanup := tc.client.Pipeline()
	for tokenID, cmd := range getCmds {
		dataJSON, err := cmd.Result()
		if err == redis.Nil {
			cleanup.SRem(ctx, key, tokenID)
			continue
		}
		if err != nil {
			continue
		}

		var data tokenData
		if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
			continue
		}

		valid = append(valid, auth.TokenInfo{
			ID:          tokenID,
			Description: data.Description,
		})
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

	dataJSON, err := tc.client.Get(ctx, tokenKey).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}

	var data tokenData
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		pipe := tc.client.TxPipeline()
		pipe.Del(ctx, tokenKey)
		pipe.SRem(ctx, tc.userTokensKey(dataJSON), tokenID)
		_, err = pipe.Exec(ctx)
		return err
	}

	pipe := tc.client.TxPipeline()
	pipe.Del(ctx, tokenKey)
	pipe.SRem(ctx, tc.userTokensKey(data.UserID), tokenID)

	_, err = pipe.Exec(ctx)
	return err
}

func (tc *tokensCache) tokenKey(tokenID string) string {
	return fmt.Sprintf("%s:token:%s", refreshPrefix, tokenID)
}

func (tc *tokensCache) userTokensKey(userID string) string {
	return fmt.Sprintf("%s:user_tokens:%s", refreshPrefix, userID)
}
