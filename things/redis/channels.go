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

const chanPrefix = "channel"

var _ things.ChannelCache = (*channelCache)(nil)

type channelCache struct {
	client *redis.Client
}

// NewChannelCache returns redis channel cache implementation.
func NewChannelCache(client *redis.Client) things.ChannelCache {
	return channelCache{client: client}
}

func (cc channelCache) Connect(ctx context.Context, chanID, thingID string) error {
	cid, tid := kv(chanID, thingID)
	if err := cc.client.SAdd(ctx, cid, tid).Err(); err != nil {
		return errors.Wrap(things.ErrConnect, err)
	}
	return nil
}

func (cc channelCache) HasThing(ctx context.Context, chanID, thingID string) bool {
	cid, tid := kv(chanID, thingID)
	return cc.client.SIsMember(ctx, cid, tid).Val()
}

func (cc channelCache) Disconnect(ctx context.Context, chanID, thingID string) error {
	cid, tid := kv(chanID, thingID)
	if err := cc.client.SRem(ctx, cid, tid).Err(); err != nil {
		return errors.Wrap(things.ErrDisconnect, err)
	}
	return nil
}

func (cc channelCache) Remove(ctx context.Context, chanID string) error {
	cid, _ := kv(chanID, "0")
	if err := cc.client.Del(ctx, cid).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

// Generates key-value pair
func kv(chanID, thingID string) (string, string) {
	cid := fmt.Sprintf("%s:%s", chanPrefix, chanID)
	return cid, thingID
}
