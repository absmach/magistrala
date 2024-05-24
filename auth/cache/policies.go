// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"strings"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

const defLimit = 100

var _ auth.Cache = (*policiesCache)(nil)

type policiesCache struct {
	client      *redis.Client
	keyDuration time.Duration
}

// NewPoliciesCache returns redis auth cache implementation.
func NewPoliciesCache(client *redis.Client, duration time.Duration) auth.Cache {
	return &policiesCache{
		client:      client,
		keyDuration: duration,
	}
}

func (pc *policiesCache) Save(ctx context.Context, key, value string) error {
	if err := pc.client.Set(ctx, key, value, pc.keyDuration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (pc *policiesCache) Contains(ctx context.Context, key, value string) bool {
	rval, err := pc.client.Get(ctx, key).Result()
	if err != nil {
		return false
	}
	if rval == value {
		return true
	}

	return false
}

func (pc *policiesCache) Remove(ctx context.Context, key string) error {
	if strings.Contains(key, "*") {
		return pc.delete(ctx, key)
	}

	if err := pc.client.Del(ctx, key).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pc *policiesCache) delete(ctx context.Context, key string) error {
	keys, cursor, err := pc.client.Scan(ctx, 0, key, defLimit).Result()
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	for cursor != 0 {
		var newKeys []string
		newKeys, cursor, err = pc.client.Scan(ctx, cursor, key, defLimit).Result()
		if err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
		keys = append(keys, newKeys...)
	}

	for _, key := range keys {
		if err := pc.client.Del(ctx, key).Err(); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
	}

	return nil
}
