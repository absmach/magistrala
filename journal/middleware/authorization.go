// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/magistrala/journal"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	mgauthz "github.com/absmach/magistrala/pkg/authz"
	"github.com/absmach/magistrala/pkg/policies"
)

var _ journal.Service = (*authorizationMiddleware)(nil)

type authorizationMiddleware struct {
	svc   journal.Service
	authz mgauthz.Authorization
}

// AuthorizationMiddleware adds authorization to the journal service.
func AuthorizationMiddleware(svc journal.Service, authz mgauthz.Authorization) journal.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) Save(ctx context.Context, journal journal.Journal) error {
	return am.svc.Save(ctx, journal)
}

func (am *authorizationMiddleware) RetrieveAll(ctx context.Context, session mgauthn.Session, page journal.Page) (journal.JournalsPage, error) {
	permission := policies.ViewPermission
	objectType := page.EntityType.AuthString()
	object := page.EntityID
	subject := session.DomainUserID

	// If the entity is a user, we need to check if the user is an admin
	if page.EntityType.AuthString() == policies.UserType {
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
	if err := am.authz.Authorize(ctx, req); err != nil {
		return journal.JournalsPage{}, err
	}

	return am.svc.RetrieveAll(ctx, session, page)
}
