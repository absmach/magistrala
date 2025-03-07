// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

type patCache struct {
	client   *redis.Client
	duration time.Duration
}

func NewPatsCache(client *redis.Client, duration time.Duration) auth.Cache {
	return &patCache{
		client:   client,
		duration: duration,
	}
}

func (pc *patCache) Save(ctx context.Context, userID string, scopes []auth.Scope) error {
	for _, sc := range scopes {
		key := generateKey(userID, sc.PatID, sc.OptionalDomainID, sc.EntityType, sc.Operation, sc.EntityID)
		if err := pc.client.Set(ctx, key, sc.ID, pc.duration).Err(); err != nil {
			return errors.Wrap(repoerr.ErrCreateEntity, err)
		}
	}

	return nil
}

func (pc *patCache) CheckScope(ctx context.Context, userID, patID, optionalDomainID string, entityType auth.EntityType, operation auth.Operation, entityID string) bool {
	exactKey := fmt.Sprintf("pat:%s:%s:%s:%s:%s:%s", userID, patID, entityType, optionalDomainID, operation, entityID)
	wildcardKey := fmt.Sprintf("pat:%s:%s:%s:%s:%s:*", userID, patID, entityType, operation, operation)

	res, err := pc.client.Exists(ctx, exactKey, wildcardKey).Result()
	if err != nil {
		return false
	}

	return res > 0
}

func (pc *patCache) Remove(ctx context.Context, userID string, scopeIDs []string) error {
	if len(scopeIDs) == 0 {
		return repoerr.ErrRemoveEntity
	}

	pattern := fmt.Sprintf("pat:%s:*", userID)
	iter := pc.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		key := iter.Val()
		val, err := pc.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}

		for _, scopeID := range scopeIDs {
			if val == scopeID {
				if err := pc.client.Del(ctx, key).Err(); err != nil {
					return errors.Wrap(repoerr.ErrRemoveEntity, err)
				}
				break
			}
		}
	}

	if err := iter.Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (pc *patCache) RemoveUserAllScope(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("pat:%s:*", userID)
	iter := pc.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := pc.client.Del(ctx, iter.Val()).Err(); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
	}
	if err := iter.Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	return nil
}

func (pc *patCache) RemoveAllScope(ctx context.Context, userID, patID string) error {
	pattern := fmt.Sprintf("pat:%s:%s", userID, patID)

	iter := pc.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := pc.client.Del(ctx, iter.Val()).Err(); err != nil {
			return errors.Wrap(repoerr.ErrRemoveEntity, err)
		}
	}

	if err := iter.Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func generateKey(userID, patID, optionalDomainId string, entityType auth.EntityType, operation auth.Operation, entityID string) string {
	return fmt.Sprintf("pat:%s:%s:%s:%s:%s:%s", userID, patID, entityType, optionalDomainId, operation, entityID)
}
