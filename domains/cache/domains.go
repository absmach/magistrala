// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"time"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/redis/go-redis/v9"
)

var (
	ErrEmptyDomainID = errors.New("domain ID is empty")
	ErrEmptyRoute    = errors.New("route is empty")
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

func (dc *domainsCache) SaveStatus(ctx context.Context, domainID string, status domains.Status) error {
	if domainID == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, ErrEmptyDomainID)
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

func (dc *domainsCache) SaveID(ctx context.Context, route, domainID string) error {
	if route == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, ErrEmptyRoute)
	}
	if domainID == "" {
		return errors.Wrap(repoerr.ErrCreateEntity, ErrEmptyDomainID)
	}
	if err := dc.client.Set(ctx, route, domainID, dc.duration).Err(); err != nil {
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

func (dc *domainsCache) ID(ctx context.Context, route string) (string, error) {
	if route == "" {
		return "", errors.Wrap(repoerr.ErrNotFound, ErrEmptyRoute)
	}
	domainID, err := dc.client.Get(ctx, route).Result()
	if err != nil {
		return "", errors.Wrap(repoerr.ErrNotFound, err)
	}

	return domainID, nil
}

func (dc *domainsCache) RemoveStatus(ctx context.Context, domainID string) error {
	if domainID == "" {
		return errors.Wrap(repoerr.ErrRemoveEntity, ErrEmptyDomainID)
	}
	if err := dc.client.Del(ctx, domainID).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}

func (dc *domainsCache) RemoveID(ctx context.Context, route string) error {
	if route == "" {
		return errors.Wrap(repoerr.ErrRemoveEntity, ErrEmptyRoute)
	}
	if err := dc.client.Del(ctx, route).Err(); err != nil {
		return errors.Wrap(repoerr.ErrRemoveEntity, err)
	}

	return nil
}
