// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"context"

	"github.com/absmach/supermq/journal"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	smqauthz "github.com/absmach/supermq/pkg/authz"
	"github.com/absmach/supermq/pkg/policies"
)

var (
	_ journal.Service = (*authorizationMiddleware)(nil)

	readPermission = "read_permission"
)

type authorizationMiddleware struct {
	svc   journal.Service
	authz smqauthz.Authorization
}

// NewAuthorization adds authorization to the journal service.
func NewAuthorization(svc journal.Service, authz smqauthz.Authorization) journal.Service {
	return &authorizationMiddleware{
		svc:   svc,
		authz: authz,
	}
}

func (am *authorizationMiddleware) Save(ctx context.Context, journal journal.Journal) error {
	return am.svc.Save(ctx, journal)
}

func (am *authorizationMiddleware) RetrieveAll(ctx context.Context, session smqauthn.Session, page journal.Page) (journal.JournalsPage, error) {
	permission := readPermission
	objectType := page.EntityType.String()
	object := page.EntityID
	subject := session.DomainUserID

	// If the entity is a user, we need to check if the user is an admin
	if page.EntityType.String() == policies.UserType {
		permission = policies.AdminPermission
		objectType = policies.PlatformType
		object = policies.SuperMQObject
		subject = session.UserID
	}

	req := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     subject,
		Permission:  permission,
		ObjectType:  objectType,
		Object:      object,
	}
	if err := am.authz.Authorize(ctx, req, nil); err != nil {
		return journal.JournalsPage{}, err
	}

	return am.svc.RetrieveAll(ctx, session, page)
}

func (am *authorizationMiddleware) RetrieveClientTelemetry(ctx context.Context, session smqauthn.Session, clientID string) (journal.ClientTelemetry, error) {
	req := smqauthz.PolicyReq{
		Domain:      session.DomainID,
		SubjectType: policies.UserType,
		SubjectKind: policies.UsersKind,
		Subject:     session.DomainUserID,
		Permission:  readPermission,
		ObjectType:  policies.ClientType,
		Object:      clientID,
	}

	if err := am.authz.Authorize(ctx, req, nil); err != nil {
		return journal.ClientTelemetry{}, err
	}

	return am.svc.RetrieveClientTelemetry(ctx, session, clientID)
}
