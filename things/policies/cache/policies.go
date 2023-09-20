// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
)

const separator = ":"

var _ policies.Cache = (*pcache)(nil)

type pcache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewCache returns redis policy cache implementation.
func NewCache(client *redis.Client, duration time.Duration) policies.Cache {
	return &pcache{
		client:      client,
		keyDuration: duration,
	}
}

func (pc *pcache) Put(ctx context.Context, policy policies.CachedPolicy) error {
	key, value := kv(policy)

	if err := pc.client.Set(ctx, key, value, pc.keyDuration).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}

	return nil
}

func (pc *pcache) Get(ctx context.Context, policy policies.CachedPolicy) (policies.CachedPolicy, error) {
	key, _ := kv(policy)
	res := pc.client.Get(ctx, key)
	// Nil response indicates non-existent key in Redis client.
	if res == nil || res.Err() == redis.Nil {
		return policies.CachedPolicy{}, errors.ErrNotFound
	}

	if err := res.Err(); err != nil {
		return policies.CachedPolicy{}, err
	}

	val, err := res.Result()
	if err != nil {
		return policies.CachedPolicy{}, err
	}

	thingID := extractThingID(val)
	if thingID == "" {
		return policies.CachedPolicy{}, errors.ErrNotFound
	}

	policy.ThingID = thingID
	policy.Actions = separateActions(val)

	return policy, nil
}

func (pc *pcache) Remove(ctx context.Context, policy policies.CachedPolicy) error {
	key, _ := kv(policy)
	if err := pc.client.Del(ctx, key).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}

	return nil
}

// kv is used to create a key-value pair for caching.
func kv(p policies.CachedPolicy) (string, string) {
	key := p.ThingKey + separator + p.ChannelID
	val := strings.Join(p.Actions, separator)

	if p.ThingID != "" {
		val += separator + p.ThingID
	}

	return key, val
}

// separateActions is used to separate the actions from the cache values.
func separateActions(actions string) []string {
	return strings.Split(actions, separator)
}

// extractThingID is used to extract the thingID from the cache values.
func extractThingID(actions string) string {
	var lastIdx = strings.LastIndex(actions, separator)

	thingID := actions[lastIdx+1:]
	// check if the thingID is a valid UUID
	if len(thingID) != 36 {
		return ""
	}

	return thingID
}
