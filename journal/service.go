// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal

import (
	"context"

	"github.com/absmach/magistrala"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

type service struct {
	idProvider magistrala.IDProvider
	repository Repository
}

func NewService(idp magistrala.IDProvider, repository Repository) Service {
	return &service{
		idProvider: idp,
		repository: repository,
	}
}

func (svc *service) Save(ctx context.Context, journal Journal) error {
	id, err := svc.idProvider.ID()
	if err != nil {
		return err
	}
	journal.ID = id

	return svc.repository.Save(ctx, journal)
}

func (svc *service) RetrieveAll(ctx context.Context, session mgauthn.Session, page Page) (JournalsPage, error) {
	journalPage, err := svc.repository.RetrieveAll(ctx, page)
	if err != nil {
		return JournalsPage{}, errors.Wrap(svcerr.ErrViewEntity, err)
	}

	return journalPage, nil
}
