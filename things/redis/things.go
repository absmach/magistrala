// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things"
)

const (
	keyPrefix = "thing_key"
	idPrefix  = "thing"
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

func (tc *thingCache) Save(ctx context.Context, thingKey string, thingID string) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	if err := tc.client.Set(ctx, tkey, thingID, 0).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	if err := tc.client.Set(ctx, tid, thingKey, 0).Err(); err != nil {
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
