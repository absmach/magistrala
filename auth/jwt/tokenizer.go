// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

const issuerName = "mainflux.auth"

type claims struct {
	jwt.StandardClaims
	IssuerID string  `json:"issuer_id,omitempty"`
	Type     *uint32 `json:"type,omitempty"`
}

func (c claims) Valid() error {
	if c.Type == nil || *c.Type > auth.APIKey || c.Issuer != issuerName {
		return errors.ErrMalformedEntity
	}

	return c.StandardClaims.Valid()
}

type tokenizer struct {
	secret string
}

// New returns new JWT Tokenizer.
func New(secret string) auth.Tokenizer {
	return tokenizer{secret: secret}
}

func (svc tokenizer) Issue(key auth.Key) (string, error) {
	claims := claims{
		StandardClaims: jwt.StandardClaims{
			Issuer:   issuerName,
			Subject:  key.Subject,
			IssuedAt: key.IssuedAt.UTC().Unix(),
		},
		IssuerID: key.IssuerID,
		Type:     &key.Type,
	}

	if !key.ExpiresAt.IsZero() {
		claims.ExpiresAt = key.ExpiresAt.UTC().Unix()
	}
	if key.ID != "" {
		claims.Id = key.ID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(svc.secret))
}

func (svc tokenizer) Parse(token string) (auth.Key, error) {
	c := claims{}
	_, err := jwt.ParseWithClaims(token, &c, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.ErrAuthentication
		}
		return []byte(svc.secret), nil
	})

	if err != nil {
		if e, ok := err.(*jwt.ValidationError); ok && e.Errors == jwt.ValidationErrorExpired {
			// Expired User key needs to be revoked.
			if c.Type != nil && *c.Type == auth.APIKey {
				return c.toKey(), auth.ErrAPIKeyExpired
			}
			return auth.Key{}, errors.Wrap(auth.ErrKeyExpired, err)
		}
		return auth.Key{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return c.toKey(), nil
}

func (c claims) toKey() auth.Key {
	key := auth.Key{
		ID:       c.Id,
		IssuerID: c.IssuerID,
		Subject:  c.Subject,
		IssuedAt: time.Unix(c.IssuedAt, 0).UTC(),
	}
	if c.ExpiresAt != 0 {
		key.ExpiresAt = time.Unix(c.ExpiresAt, 0).UTC()
	}

	// Default type is 0.
	if c.Type != nil {
		key.Type = *c.Type
	}

	return key
}
