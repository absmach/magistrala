// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/redis/go-redis/v9"
)

const (
	refreshPrefix = "refresh_tokens:"
	scoreNegInf   = "-inf"
	scorePosInf   = "+inf"
)

type tokenData struct {
	UserID      string `json:"user_id"`
	Description string `json:"description,omitempty"`
}

var _ auth.UserActiveTokensCache = (*tokensCache)(nil)

type tokensCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewUserActiveTokensCache returns redis auth cache implementation.
func NewUserActiveTokensCache(client *redis.Client, duration time.Duration) (auth.UserActiveTokensCache, error) {
	if duration == 0 {
		return nil, errors.New("token cache duration must not be zero")
	}
	return &tokensCache{
		client:      client,
		keyDuration: duration,
	}, nil
}

// SaveActive saves an active refresh token ID for a user with optional description.
func (tc *tokensCache) SaveActive(ctx context.Context, userID, tokenID, description string, expiry time.Time) error {
	ttl := min(tc.keyDuration, time.Until(expiry))

	data := tokenData{
		UserID:      userID,
		Description: description,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	pipe := tc.client.TxPipeline()

	pipe.Set(ctx, tokenKey(tokenID), dataJSON, ttl)
	pipe.ZAdd(ctx, userTokensKey(userID), redis.Z{
		Score:  float64(expiry.Unix()),
		Member: tokenID,
	})

	_, err = pipe.Exec(ctx)

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
	getCmds := make(map[string]*redis.StringCmd, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		getCmds[tokenID] = getPipe.Get(ctx, tokenKey(tokenID))
	}

	if _, err = getPipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}

	valid := make([]auth.TokenInfo, 0, len(tokenIDs))
	for tokenID, cmd := range getCmds {
		dataJSON, err := cmd.Result()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
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

	return valid, nil
}

// RemoveActive removes an active refresh token ID for a user.
func (tc *tokensCache) RemoveActive(ctx context.Context, tokenID string) (err error) {
	tokenKeyStr := tokenKey(tokenID)

	dataJSON, err := tc.client.Get(ctx, tokenKeyStr).Result()
	if err == redis.Nil {
		return svcerr.ErrNotFound
	}
	if err != nil {
		return err
	}

	pipe := tc.client.TxPipeline()
	pipe.Del(ctx, tokenKeyStr)
	defer func() {
		_, execErr := pipe.Exec(ctx)
		if err == nil {
			err = execErr
		}
	}()

	var data tokenData
	if err = json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return err
	}

	pipe.ZRem(ctx, userTokensKey(data.UserID), tokenID)
	return nil
}

func tokenKey(tokenID string) string {
	return refreshPrefix + "token:" + tokenID
}

func userTokensKey(userID string) string {
	return refreshPrefix + "user_tokens:" + userID
}
