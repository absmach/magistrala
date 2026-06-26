// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package atom

import (
	"context"

	atomcore "github.com/absmach/magistrala/internal/atom"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
)

type authentication struct {
	verifier *atomcore.TokenVerifier
}

var _ authn.Authentication = (*authentication)(nil)

func NewAuthentication() authn.Authentication {
	return authentication{verifier: atomcore.NewTokenVerifier(atomcore.LoadConfig())}
}

func (a authentication) Authenticate(ctx context.Context, token string) (authn.Session, error) {
	claims, err := a.verifier.VerifyTokenClaims(ctx, token)
	if err != nil {
		return authn.Session{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	return authn.Session{
		Type:     authn.AccessToken,
		UserID:   claims.SubjectID,
		DomainID: claims.TenantID,
		Role:     authn.UserRole,
		Verified: true,
	}, nil
}
