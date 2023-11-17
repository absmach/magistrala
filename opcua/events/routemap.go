// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"
	"fmt"

	"github.com/absmach/magistrala/opcua"
	"github.com/go-redis/redis/v8"
)

var _ opcua.RouteMapRepository = (*routerMap)(nil)

type routerMap struct {
	client *redis.Client
	prefix string
}

// NewRouteMapRepository returns redis thing cache implementation.
func NewRouteMapRepository(client *redis.Client, prefix string) opcua.RouteMapRepository {
	return &routerMap{
		client: client,
		prefix: prefix,
	}
}

func (mr *routerMap) Save(ctx context.Context, mgxID, opcuaID string) error {
	tkey := fmt.Sprintf("%s:%s", mr.prefix, mgxID)
	if err := mr.client.Set(ctx, tkey, opcuaID, 0).Err(); err != nil {
		return err
	}

	lkey := fmt.Sprintf("%s:%s", mr.prefix, opcuaID)
	if err := mr.client.Set(ctx, lkey, mgxID, 0).Err(); err != nil {
		return err
	}

	return nil
}

func (mr *routerMap) Get(ctx context.Context, opcuaID string) (string, error) {
	lKey := fmt.Sprintf("%s:%s", mr.prefix, opcuaID)
	mval, err := mr.client.Get(ctx, lKey).Result()
	if err != nil {
		return "", err
	}

	return mval, nil
}

func (mr *routerMap) Remove(ctx context.Context, mgxID string) error {
	mkey := fmt.Sprintf("%s:%s", mr.prefix, mgxID)
	lval, err := mr.client.Get(ctx, mkey).Result()
	if err != nil {
		return err
	}

	lkey := fmt.Sprintf("%s:%s", mr.prefix, lval)
	return mr.client.Del(ctx, mkey, lkey).Err()
}
