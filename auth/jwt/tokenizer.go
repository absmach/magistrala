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
	jwt.RegisteredClaims
	IssuerID string  `json:"issuer_id,omitempty"`
	Type     *uint32 `json:"type,omitempty"`
}

func (c claims) Valid() error {
	if c.Type == nil || *c.Type > auth.APIKey || c.Issuer != issuerName {
		return errors.ErrMalformedEntity
	}

	return c.RegisteredClaims.Valid()
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
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   issuerName,
			Subject:  key.Subject,
			IssuedAt: &jwt.NumericDate{Time: key.IssuedAt.UTC()},
		},
		IssuerID: key.IssuerID,
		Type:     &key.Type,
	}

	if !key.ExpiresAt.IsZero() {
		claims.ExpiresAt = &jwt.NumericDate{Time: key.ExpiresAt.UTC()}
	}
	if key.ID != "" {
		claims.ID = key.ID
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
		ID:       c.ID,
		IssuerID: c.IssuerID,
		Subject:  c.Subject,
		IssuedAt: c.IssuedAt.Time.UTC(),
	}

	key.ExpiresAt = time.Time{}
	if c.ExpiresAt != nil && c.ExpiresAt.Time.UTC().Unix() != 0 {
		key.ExpiresAt = c.ExpiresAt.Time.UTC()
	}

	// Default type is 0.
	if c.Type != nil {
		key.Type = *(c.Type)
	}

	return key
}
