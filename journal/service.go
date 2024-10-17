// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package journal

import (
	"context"

	"github.com/absmach/magistrala"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/policies"
)

type service struct {
	authn      mgauthn.Authentication
	authz      mgauthz.Authorization
	idProvider magistrala.IDProvider
	repository Repository
}

func NewService(authn mgauthn.Authentication, authz mgauthz.Authorization, idp magistrala.IDProvider, repository Repository) Service {
	return &service{
		idProvider: idp,
		authn:      authn,
		authz:      authz,
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

func (svc *service) RetrieveAll(ctx context.Context, token string, page Page) (JournalsPage, error) {
	if err := svc.authorize(ctx, token, page.EntityID, page.EntityType.AuthString()); err != nil {
		return JournalsPage{}, err
	}

	return svc.repository.RetrieveAll(ctx, page)
}

func (svc *service) authorize(ctx context.Context, token, entityID, entityType string) error {
	session, err := svc.authn.Authenticate(ctx, token)
	if err != nil {
		return err
	}

	permission := policies.ViewPermission
	objectType := entityType
	object := entityID
	subject := session.DomainUserID

	// If the entity is a user, we need to check if the user is an admin
	if entityType == policies.UserType {
		permission = policies.AdminPermission
		objectType = policies.PlatformType
		object = policies.MagistralaObject
		subject = session.UserID
	}

	req := mgauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subject,
		Permission:  permission,
		ObjectType:  objectType,
		Object:      object,
	}

	if err := svc.authz.Authorize(ctx, req); err != nil {
		return err
	}

	return nil
}
