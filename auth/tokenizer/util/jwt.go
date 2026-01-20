// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var (
	// ErrInvalidIssuer represents an invalid token issuer value.
	ErrInvalidIssuer = errors.New("invalid token issuer value")

	// ErrJSONHandle indicates an error in handling JSON.
	ErrJSONHandle = errors.New("failed to perform operation JSON")

	// ErrJWTExpiryKey indicates that the "exp" claim in the JWT token is not satisfied.
	ErrJWTExpiryKey = errors.New(`"exp" not satisfied`)

	errInvalidType     = errors.New("invalid token type")
	errInvalidRole     = errors.New("invalid role")
	errInvalidVerified = errors.New("invalid verified")
)

const (
	IssuerName    = "supermq.auth"
	TokenType     = "type"
	RoleField     = "role"
	VerifiedField = "verified"
	PatPrefix     = "pat"
)

// ToKey converts a JWT token to an auth.Key by extracting claims.
func ToKey(tkn jwt.Token) (auth.Key, error) {
	data, err := json.Marshal(tkn.PrivateClaims())
	if err != nil {
		return auth.Key{}, errors.Wrap(ErrJSONHandle, err)
	}
	var key auth.Key
	if err := json.Unmarshal(data, &key); err != nil {
		return auth.Key{}, errors.Wrap(ErrJSONHandle, err)
	}

	tType, ok := tkn.Get(TokenType)
	if !ok {
		return auth.Key{}, errInvalidType
	}
	kType, ok := tType.(float64)
	if !ok {
		return auth.Key{}, errInvalidType
	}
	kt := auth.KeyType(kType)
	if !kt.Validate() {
		return auth.Key{}, errInvalidType
	}

	tRole, ok := tkn.Get(RoleField)
	if !ok {
		return auth.Key{}, errInvalidRole
	}
	kRole, ok := tRole.(float64)
	if !ok {
		return auth.Key{}, errInvalidRole
	}

	tVerified, ok := tkn.Get(VerifiedField)
	if !ok {
		return auth.Key{}, errInvalidVerified
	}
	kVerified, ok := tVerified.(bool)
	if !ok {
		return auth.Key{}, errInvalidVerified
	}

	kr := auth.Role(kRole)
	if !kr.Validate() {
		return auth.Key{}, errInvalidRole
	}

	key.ID = tkn.JwtID()
	key.Type = auth.KeyType(kType)
	key.Role = auth.Role(kRole)
	key.Issuer = tkn.Issuer()
	key.Subject = tkn.Subject()
	key.IssuedAt = tkn.IssuedAt()
	key.ExpiresAt = tkn.Expiration()
	key.Verified = kVerified

	return key, nil
}

func BuildToken(key auth.Key) (jwt.Token, error) {
	builder := jwt.NewBuilder()
	builder.
		Issuer(IssuerName).
		IssuedAt(key.IssuedAt).
		Claim(TokenType, key.Type).
		Expiration(key.ExpiresAt).
		Claim(RoleField, key.Role).
		Claim(VerifiedField, key.Verified)

	if key.Subject != "" {
		builder.Subject(key.Subject)
	}
	if key.ID != "" {
		builder.JwtID(key.ID)
	}

	return builder.Build()
}
