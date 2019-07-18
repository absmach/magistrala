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

const chanPrefix = "channel"

var _ things.ChannelCache = (*channelCache)(nil)

type channelCache struct {
	client *redis.Client
}

// NewChannelCache returns redis channel cache implementation.
func NewChannelCache(client *redis.Client) things.ChannelCache {
	return channelCache{client: client}
}

func (cc channelCache) Connect(_ context.Context, chanID, thingID string) error {
	cid, tid := kv(chanID, thingID)
	return cc.client.SAdd(cid, tid).Err()
}

func (cc channelCache) HasThing(_ context.Context, chanID, thingID string) bool {
	cid, tid := kv(chanID, thingID)
	return cc.client.SIsMember(cid, tid).Val()
}

func (cc channelCache) Disconnect(_ context.Context, chanID, thingID string) error {
	cid, tid := kv(chanID, thingID)
	return cc.client.SRem(cid, tid).Err()
}

func (cc channelCache) Remove(_ context.Context, chanID string) error {
	cid, _ := kv(chanID, "0")
	return cc.client.Del(cid).Err()
}

// Generates key-value pair
func kv(chanID, thingID string) (string, string) {
	cid := fmt.Sprintf("%s:%s", chanPrefix, chanID)
	return cid, thingID
}
