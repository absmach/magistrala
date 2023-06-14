// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"
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
	return pcache{
		client:      client,
		keyDuration: duration,
	}
}

func (pc pcache) Put(ctx context.Context, policy policies.Policy) error {
	k, v := kv(policy)
	if err := pc.client.Set(ctx, k, v, pc.keyDuration).Err(); err != nil {
		return errors.Wrap(errors.ErrCreateEntity, err)
	}
	return nil
}

func (pc pcache) Get(ctx context.Context, policy policies.Policy) (policies.Policy, error) {
	k, _ := kv(policy)
	res := pc.client.Get(ctx, k)
	// Nil response indicates non-existent key in Redis client.
	if res == nil || res.Err() == redis.Nil {
		return policies.Policy{}, errors.ErrNotFound
	}
	if err := res.Err(); err != nil {
		return policies.Policy{}, err
	}
	actions, err := res.Result()
	if err != nil {
		return policies.Policy{}, err
	}
	policy.Actions = strings.Split(actions, separator)
	return policy, nil
}

func (pc pcache) Remove(ctx context.Context, policy policies.Policy) error {
	obj, _ := kv(policy)
	if err := pc.client.Del(ctx, obj).Err(); err != nil {
		return errors.Wrap(errors.ErrRemoveEntity, err)
	}
	return nil
}

// Generates key-value pair for Redis client.
func kv(p policies.Policy) (string, string) {
	return fmt.Sprintf("%s%s%s", p.Subject, separator, p.Object), strings.Join(p.Actions, separator)
}
