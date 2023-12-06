// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/things"
	"github.com/go-redis/redis/v8"
)

const (
	keyPrefix = "thing_key"
	idPrefix  = "thing_id"
)

var _ things.Cache = (*thingCache)(nil)

type thingCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewCache returns redis thing cache implementation.
func NewCache(client *redis.Client, duration time.Duration) things.Cache {
	return &thingCache{
		client:      client,
		keyDuration: duration,
	}
}

func (tc *thingCache) Save(ctx context.Context, thingKey, thingID string) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	if err := tc.client.Set(ctx, tkey, thingID, tc.keyDuration).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	if err := tc.client.Set(ctx, tid, thingKey, tc.keyDuration).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (tc *thingCache) ID(ctx context.Context, thingKey string) (string, error) {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	thingID, err := tc.client.Get(ctx, tkey).Result()
	if err != nil {
		return "", errors.Wrap(errors.ErrNotFound, err)
	}
	if thingID == "" {
		return "", errors.ErrNotFound
	}

	return thingID, nil
}

func (tc *thingCache) Remove(ctx context.Context, thingID string) error {
	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	key, err := tc.client.Get(ctx, tid).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)
	if err := tc.client.Del(ctx, tkey, tid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}
