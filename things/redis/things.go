// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
)

const (
	keyPrefix = "thing_key"
	idPrefix  = "thing"
)

var (
	// ErrRedisThingSave indicates error while saving Thing in redis cache
	ErrRedisThingSave = errors.New("failed to save thing in redis cache")

	// ErrRedisThingID indicates error while geting Thing ID from redis cache
	ErrRedisThingID = errors.New("failed to get thing id from redis cache")

	// ErrRedisThingRemove indicates error while removing Thing from redis cache
	ErrRedisThingRemove = errors.New("failed to remove thing from redis cache")
)

var _ things.ThingCache = (*thingCache)(nil)

type thingCache struct {
	client *redis.Client
}

// NewThingCache returns redis thing cache implementation.
func NewThingCache(client *redis.Client) things.ThingCache {
	return &thingCache{
		client: client,
	}
}

func (tc *thingCache) Save(_ context.Context, thingKey string, thingID string) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	if err := tc.client.Set(tkey, thingID, 0).Err(); err != nil {
		return errors.Wrap(ErrRedisThingSave, err)
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	if err := tc.client.Set(tid, thingKey, 0).Err(); err != nil {
		return errors.Wrap(ErrRedisThingSave, err)
	}
	return nil
}

func (tc *thingCache) ID(_ context.Context, thingKey string) (string, error) {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	thingID, err := tc.client.Get(tkey).Result()
	if err != nil {
		return "", errors.Wrap(ErrRedisThingID, err)
	}

	return thingID, nil
}

func (tc *thingCache) Remove(_ context.Context, thingID string) error {
	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	key, err := tc.client.Get(tid).Result()
	// Redis returns Nil Reply when key does not exist.
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return errors.Wrap(ErrRedisThingRemove, err)
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)
	if err := tc.client.Del(tkey, tid).Err(); err != nil {
		return errors.Wrap(ErrRedisThingRemove, err)
	}
	return nil
}
