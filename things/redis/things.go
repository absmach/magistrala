//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis

import (
	"fmt"
	"strconv"

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

func (tc *thingCache) Save(thingKey string, thingID uint64) error {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	id := strconv.FormatUint(thingID, 10)
	if err := tc.client.Set(tkey, id, 0).Err(); err != nil {
		return err
	}

	tid := fmt.Sprintf("%s:%s", idPrefix, id)
	return tc.client.Set(tid, thingKey, 0).Err()
}

func (tc *thingCache) ID(thingKey string) (uint64, error) {
	tkey := fmt.Sprintf("%s:%s", keyPrefix, thingKey)
	thingID, err := tc.client.Get(tkey).Result()
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseUint(thingID, 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (tc *thingCache) Remove(thingID uint64) error {
	tid := fmt.Sprintf("%s:%d", idPrefix, thingID)
	key, err := tc.client.Get(tid).Result()
	if err != nil {
		return err
	}

	tkey := fmt.Sprintf("%s:%s", keyPrefix, key)

	return tc.client.Del(tkey, tid).Err()
}
