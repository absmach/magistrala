// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"context"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mainflux/mainflux/pkg/errors"
)

const issuerName = "clients.auth"

var _ Repository = (*tokenRepo)(nil)

type tokenRepo struct {
	secret          []byte
	accessDuration  time.Duration
	refreshDuration time.Duration
}

// NewRepository instantiates an implementation of Token repository.
func NewRepository(secret []byte, aduration, rduration time.Duration) Repository {
	return &tokenRepo{
		secret:          secret,
		accessDuration:  aduration,
		refreshDuration: rduration,
	}
}

func (repo tokenRepo) Issue(ctx context.Context, claim Claims) (Token, error) {
	aexpiry := time.Now().Add(repo.accessDuration)
	accessToken, err := jwt.NewBuilder().
		Issuer(issuerName).
		IssuedAt(time.Now()).
		Subject(claim.ClientID).
		Claim("identity", claim.Email).
		Claim("type", AccessToken).
		Expiration(aexpiry).
		Build()
	if err != nil {
		return Token{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	signedAccessToken, err := jwt.Sign(accessToken, jwt.WithKey(jwa.HS512, repo.secret))
	if err != nil {
		return Token{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	refreshToken, err := jwt.NewBuilder().
		Issuer(issuerName).
		IssuedAt(time.Now()).
		Subject(claim.ClientID).
		Claim("identity", claim.Email).
		Claim("type", RefreshToken).
		Expiration(time.Now().Add(repo.refreshDuration)).
		Build()
	if err != nil {
		return Token{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	signedRefreshToken, err := jwt.Sign(refreshToken, jwt.WithKey(jwa.HS512, repo.secret))
	if err != nil {
		return Token{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return Token{
		AccessToken:  string(signedAccessToken),
		RefreshToken: string(signedRefreshToken),
		AccessType:   "Bearer",
	}, nil
}

func (repo tokenRepo) Parse(ctx context.Context, accessToken string) (Claims, error) {
	token, err := jwt.Parse(
		[]byte(accessToken),
		jwt.WithValidate(true),
		jwt.WithKey(jwa.HS512, repo.secret),
	)
	if err != nil {
		return Claims{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	tType, ok := token.Get("type")
	if !ok {
		return Claims{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	identity, ok := token.Get("identity")
	if !ok {
		return Claims{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	claim := Claims{
		ClientID: token.Subject(),
		Email:    identity.(string),
		Type:     tType.(string),
	}
	return claim, nil
}
