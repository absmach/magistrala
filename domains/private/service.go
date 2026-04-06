// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/magistrala/domains"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

const defLimit = 100

type Service interface {
	RetrieveStatus(ctx context.Context, id string) (domains.Status, error)
	DeleteUserFromDomains(ctx context.Context, id string) error
	RetrieveIDByRoute(ctx context.Context, route string) (string, error)
}

var _ Service = (*service)(nil)

func New(repo domains.Repository, cache domains.Cache) Service {
	return service{
		repo:  repo,
		cache: cache,
	}
}

type service struct {
	repo  domains.Repository
	cache domains.Cache
}

func (svc service) RetrieveStatus(ctx context.Context, id string) (domains.Status, error) {
	status, err := svc.cache.Status(ctx, id)
	if err == nil {
		return status, nil
	}
	dom, err := svc.repo.RetrieveDomainByID(ctx, id)
	if err != nil {
		return domains.AllStatus, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	status = dom.Status
	if err := svc.cache.SaveStatus(ctx, id, status); err != nil {
		return domains.AllStatus, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dom.Status, nil
}

func (svc service) DeleteUserFromDomains(ctx context.Context, id string) (err error) {
	domainsPage, err := svc.repo.ListDomains(ctx, domains.Page{UserID: id, Limit: defLimit})
	if err != nil {
		return err
	}

	if domainsPage.Total > defLimit {
		for i := defLimit; i < int(domainsPage.Total); i += defLimit {
			page := domains.Page{UserID: id, Offset: uint64(i), Limit: defLimit}
			dp, err := svc.repo.ListDomains(ctx, page)
			if err != nil {
				return err
			}
			domainsPage.Domains = append(domainsPage.Domains, dp.Domains...)
		}
	}

	return nil
}

func (svc service) RetrieveIDByRoute(ctx context.Context, route string) (string, error) {
	id, err := svc.cache.ID(ctx, route)
	if err == nil {
		return id, nil
	}
	dom, err := svc.repo.RetrieveDomainByRoute(ctx, route)
	if err != nil {
		return "", errors.Wrap(svcerr.ErrViewEntity, err)
	}
	if err := svc.cache.SaveID(ctx, route, dom.ID); err != nil {
		return "", errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return dom.ID, nil
}
