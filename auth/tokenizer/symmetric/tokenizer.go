// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package symmetric

import (
	"context"
	"time"

	"github.com/absmach/supermq/auth"
	smqjwt "github.com/absmach/supermq/auth/tokenizer/util"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type tokenizer struct {
	algorithm jwa.KeyAlgorithm
	secret    []byte
	cache     auth.TokensCache
}

var _ auth.Tokenizer = (*tokenizer)(nil)

func NewTokenizer(algorithm string, secret []byte, cache auth.TokensCache) (auth.Tokenizer, error) {
	alg := jwa.KeyAlgorithmFrom(algorithm)
	if _, ok := alg.(jwa.InvalidKeyAlgorithm); ok {
		return nil, auth.ErrUnsupportedKeyAlgorithm
	}
	if len(secret) == 0 {
		return nil, auth.ErrInvalidSymmetricKey
	}
	return &tokenizer{
		secret:    secret,
		algorithm: alg,
		cache:     cache,
	}, nil
}

func (tok *tokenizer) Issue(ctx context.Context, key auth.Key) (string, error) {
	tkn, err := smqjwt.BuildToken(key)
	if err != nil {
		return "", err
	}

	signedBytes, err := jwt.Sign(tkn, jwt.WithKey(tok.algorithm, tok.secret))
	if err != nil {
		return "", err
	}

	// Store refresh tokens as active with TTL
	if key.Type == auth.RefreshKey && key.ID != "" && key.Subject != "" {
		ttl := time.Until(key.ExpiresAt)
		if ttl > 0 {
			if err := tok.cache.SaveActive(ctx, key.Subject, key.ID, key.Description, ttl); err != nil {
				return "", err
			}
		}
	}

	return string(signedBytes), nil
}

func (tok *tokenizer) Parse(ctx context.Context, tokenString string) (auth.Key, error) {
	key, err := tok.parseToken(tokenString)
	if err != nil {
		return auth.Key{}, err
	}
	if key.Type == auth.RefreshKey {
		// Check if the refresh token is active for this user
		found, err := tok.cache.IsActive(ctx, key.ID)
		if err != nil {
			return auth.Key{}, err
		}
		if !found {
		}
	}

	return key, nil
}

func (tok *tokenizer) RetrieveJWKS() ([]auth.PublicKeyInfo, error) {
	return nil, auth.ErrPublicKeysNotSupported
}

func (tok *tokenizer) Revoke(ctx context.Context, token string) error {
	key, err := tok.parseToken(token)
	if err != nil {
		return err
	}

	if key.Type == auth.RefreshKey {
		// Remove the refresh token from active tokens
		if err := tok.cache.RemoveActive(ctx, key.ID); err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}
	}

	return nil
}

func (tok *tokenizer) parseToken(tokenString string) (auth.Key, error) {
	if len(tokenString) >= 3 && tokenString[:3] == smqjwt.PatPrefix {
		return auth.Key{Type: auth.PersonalAccessToken}, nil
	}

	tkn, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithValidate(true),
		jwt.WithKey(tok.algorithm, tok.secret),
	)
	if err != nil {
		if errors.Contains(err, smqjwt.ErrJWTExpiryKey) {
			return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, auth.ErrExpiry)
		}
		return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if tkn.Issuer() != smqjwt.IssuerName {
		return auth.Key{}, smqjwt.ErrInvalidIssuer
	}

	return smqjwt.ToKey(tkn)
}
