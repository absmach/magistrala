// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package eventlogs

import (
	"context"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

type service struct {
	auth       magistrala.AuthServiceClient
	repository Repository
}

func NewService(repository Repository, authClient magistrala.AuthServiceClient) Service {
	return &service{
		auth:       authClient,
		repository: repository,
	}
}

func (svc *service) ReadAll(ctx context.Context, token string, page Page) (EventsPage, error) {
	if err := svc.authorize(ctx, token, page.ID, page.EntityType); err != nil {
		return EventsPage{}, err
	}

	return svc.repository.RetrieveAll(ctx, page)
}

func (svc *service) authorize(ctx context.Context, token, id, entityType string) error {
	req := &magistrala.AuthorizeReq{
		SubjectType: auth.UserType,
		SubjectKind: auth.TokenKind,
		Subject:     token,
		Permission:  auth.ViewPermission,
		ObjectType:  entityType,
		Object:      id,
	}

	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}

	return nil
}
