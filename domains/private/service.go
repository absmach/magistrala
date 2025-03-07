// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package private

import (
	"context"

	"github.com/absmach/supermq/domains"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
)

const defLimit = 100

//go:generate mockery --name Service  --output=./mocks --filename service.go --quiet --note "Copyright (c) Abstract Machines"
type Service interface {
	RetrieveEntity(ctx context.Context, id string) (domains.Domain, error)
	DeleteUserFromDomains(ctx context.Context, id string) error
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

func (svc service) RetrieveEntity(ctx context.Context, id string) (domains.Domain, error) {
	status, err := svc.cache.Status(ctx, id)
	if err == nil {
		return domains.Domain{ID: id, Status: status}, nil
	}
	dom, err := svc.repo.RetrieveDomainByID(ctx, id)
	if err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}
	status = dom.Status
	if err := svc.cache.Save(ctx, id, status); err != nil {
		return domains.Domain{}, errors.Wrap(svcerr.ErrUpdateEntity, err)
	}

	return domains.Domain{ID: dom.ID, Status: dom.Status}, nil
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
