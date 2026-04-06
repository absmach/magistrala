// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	"github.com/redis/go-redis/v9"
)

var (
	ErrEmptyDomainID     = errors.New("domain ID is empty")
	ErrEmptyChannelID    = errors.New("channel ID is empty")
	ErrEmptyChannelRoute = errors.New("channel route is empty")
)

type channelsCache struct {
	client   *redis.Client
	duration time.Duration
}

func NewChannelsCache(client *redis.Client, duration time.Duration) channels.Cache {
	return &channelsCache{
		client:   client,
		duration: duration,
	}
}

func (cc *channelsCache) Save(ctx context.Context, route, domainID, channelID string) error {
	key, err := encodeKey(domainID, route)
	if err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}
	if channelID == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, ErrEmptyChannelID)
	}
	if err := cc.client.Set(ctx, key, channelID, cc.duration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (cc *channelsCache) ID(ctx context.Context, channelRoute, domainID string) (string, error) {
	key, err := encodeKey(domainID, channelRoute)
	if err != nil {
		return "", errors.Wrap(repoerr.ErrNotFound, err)
	}
	id, err := cc.client.Get(ctx, key).Result()
	if err != nil {
		return "", errors.Wrap(repoerr.ErrNotFound, err)
	}

	return id, nil
}

func (cc *channelsCache) Remove(ctx context.Context, channelRoute, domainID string) error {
	key, err := encodeKey(domainID, channelRoute)
	if err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}
	if err := cc.client.Del(ctx, key).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func encodeKey(domainID, channelRoute string) (string, error) {
	if domainID == "" {
		return "", ErrEmptyDomainID
	}
	if channelRoute == "" {
		return "", ErrEmptyChannelRoute
	}
	return domainID + ":" + channelRoute, nil
}
