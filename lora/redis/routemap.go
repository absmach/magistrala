//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis

import (
	"fmt"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux/lora"
)

const (
	mfxMapPrefix  = "mfx:lora"
	loraMapPrefix = "lora:mfx"
)

var _ lora.RouteMapRepository = (*routerMap)(nil)

type routerMap struct {
	client *redis.Client
	prefix string
}

// NewRouteMapRepository returns redis thing cache implementation.
func NewRouteMapRepository(client *redis.Client, prefix string) lora.RouteMapRepository {
	return &routerMap{
		client: client,
		prefix: prefix,
	}
}

func (mr *routerMap) Save(mfxID, loraID string) error {
	tkey := fmt.Sprintf("%s:%s:%s", mr.prefix, mfxMapPrefix, mfxID)
	if err := mr.client.Set(tkey, loraID, 0).Err(); err != nil {
		return err
	}
	lkey := fmt.Sprintf("%s:%s:%s", mr.prefix, loraMapPrefix, loraID)
	if err := mr.client.Set(lkey, mfxID, 0).Err(); err != nil {
		return err
	}

	return nil
}

func (mr *routerMap) Get(mfxID string) (string, error) {
	lKey := fmt.Sprintf("%s:%s:%s", mr.prefix, loraMapPrefix, mfxID)
	mval, err := mr.client.Get(lKey).Result()
	if err != nil {
		return "", err
	}

	return mval, nil
}

func (mr *routerMap) Remove(mfxID string) error {
	mkey := fmt.Sprintf("%s:%s:%s", mr.prefix, mfxMapPrefix, mfxID)
	lval, err := mr.client.Get(mkey).Result()
	if err != nil {
		return err
	}

	lkey := fmt.Sprintf("%s:%s:%s", mr.prefix, loraMapPrefix, lval)
	return mr.client.Del(mkey, lkey).Err()
}
