// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "client_key"
	idPrefix  = "client_id"
)

var _ clients.Cache = (*clientCache)(nil)

type clientCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewCache returns redis client cache implementation.
func NewCache(client *redis.Client, duration time.Duration) clients.Cache {
	return &clientCache{
		client:      client,
		keyDuration: duration,
	}
}

func (tc *clientCache) Save(ctx context.Context, clientKey, clientID string) error {
	if clientKey == "" || clientID == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, errors.New("client key or client id is empty"))
	}
	tkey := fmt.Sprintf("%s:%s", keyPrefix, clientKey)
	if err := tc.client.Set(ctx, tkey, clientID, tc.keyDuration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, clientID)
	if err := tc.client.Set(ctx, tid, clientKey, tc.keyDuration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (tc *clientCache) ID(ctx context.Context, clientKey string) (string, error) {
	if clientKey == "" {
		return "", repoerr.ErrNotFound
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, clientKey)
	clientID, err := tc.client.Get(ctx, tkey).Result()
	if err != nil {
		return "", errors.Wrap(repoerr.ErrNotFound, err)
	}

	return clientID, nil
}

func (tc *clientCache) Remove(ctx context.Context, clientID string) error {
	tid := fmt.Sprintf("%s:%s", idPrefix, clientID)
	key, err := tc.client.Get(ctx, tid).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)
	if err := tc.client.Del(ctx, tkey, tid).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}
