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
	if err := cc.client.SAdd(cid, tid).Err(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}
	return nil
}

func (cc channelCache) HasThing(_ context.Context, chanID, thingID string) bool {
	cid, tid := kv(chanID, thingID)
	return cc.client.SIsMember(cid, tid).Val()
}

func (cc channelCache) Disconnect(_ context.Context, chanID, thingID string) error {
	cid, tid := kv(chanID, thingID)
	if err := cc.client.SRem(cid, tid).Err(); err != nil {
		return errors.Wrap(things.ErrDisconnect, err)
	}
	return nil
}

func (cc channelCache) Remove(_ context.Context, chanID string) error {
	cid, _ := kv(chanID, "0")
	if err := cc.client.Del(cid).Err(); err != nil {
		return errors.Wrap(things.ErrRemoveEntity, err)
	}
	return nil
}

// Generates key-value pair
func kv(chanID, thingID string) (string, string) {
	cid := fmt.Sprintf("%s:%s", chanPrefix, chanID)
	return cid, thingID
}
