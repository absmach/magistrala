//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis"
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

func (tc *thingCache) Save(_ context.Context, thingKey string, thingID string) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	if err := tc.client.Set(tkey, thingID, 0).Err(); err != nil {
		return err
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	return tc.client.Set(tid, thingKey, 0).Err()
}

func (tc *thingCache) ID(_ context.Context, thingKey string) (string, error) {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	thingID, err := tc.client.Get(tkey).Result()
	if err != nil {
		return "", err
	}

	return thingID, nil
}

func (tc *thingCache) Remove(_ context.Context, thingID string) error {
	tid := fmt.Sprintf("%s:%s", idPrefix, thingID)
	key, err := tc.client.Get(tid).Result()
	if err != nil {
		return err
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)

	return tc.client.Del(tkey, tid).Err()
}
