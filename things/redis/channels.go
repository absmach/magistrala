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

const chanPrefix = "channel"

var _ things.ChannelCache = (*channelCache)(nil)

type channelCache struct {
	client *redis.Client
}

// NewChannelCache returns redis channel cache implementation.
func NewChannelCache(client *redis.Client) things.ChannelCache {
	return channelCache{client: client}
}

func (cc channelCache) Connect(chanID, thingID uint64) error {
	cid, tid := kv(chanID, thingID)
	return cc.client.SAdd(cid, tid).Err()
}

func (cc channelCache) HasThing(chanID, thingID uint64) bool {
	cid, tid := kv(chanID, thingID)
	return cc.client.SIsMember(cid, tid).Val()
}

func (cc channelCache) Disconnect(chanID, thingID uint64) error {
	cid, tid := kv(chanID, thingID)
	return cc.client.SRem(cid, tid).Err()
}

func (cc channelCache) Remove(chanID uint64) error {
	cid, _ := kv(chanID, 0)
	return cc.client.Del(cid).Err()
}

// Generates key-value pair
func kv(chanID, thingID uint64) (string, string) {
	cid := fmt.Sprintf("%s:%d", chanPrefix, chanID)
	tid := strconv.FormatUint(thingID, 10)
	return cid, tid
}
