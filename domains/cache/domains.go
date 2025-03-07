// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/redis/go-redis/v9"
)

type domainsCache struct {
	client   *redis.Client
	duration time.Duration
}

func NewDomainsCache(client *redis.Client, duration time.Duration) domains.Cache {
	return &domainsCache{
		client:   client,
		duration: duration,
	}
}

func (dc *domainsCache) Save(ctx context.Context, domainID string, status domains.Status) error {
	if domainID == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, errors.New("domain ID is empty"))
	}
	statusString := status.String()
	if statusString == domains.Unknown {
		return errors.Wrap(repoerr.ErrCreateEntity, svcerr.ErrInvalidStatus)
	}
	if err := dc.client.Set(ctx, domainID, status.String(), dc.duration).Err(); err != nil {
		return errors.Wrap(repoerr.ErrCreateEntity, err)
	}

	return nil
}

func (dc *domainsCache) Status(ctx context.Context, domainID string) (domains.Status, error) {
	st, err := dc.client.Get(ctx, domainID).Result()
	if err != nil {
		return domains.AllStatus, errors.Wrap(repoerr.ErrNotFound, err)
	}
	status, err := domains.ToStatus(st)
	if err != nil {
		return domains.AllStatus, errors.Wrap(repoerr.ErrNotFound, err)
	}

	return status, nil
}

func (dc *domainsCache) Remove(ctx context.Context, domainID string) error {
	if domainID == "" {
		return errors.Wrap(repoerr.ErrRemoveEntity, errors.New("domain ID is empty"))
	}
	if err := dc.client.Del(ctx, domainID).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}
