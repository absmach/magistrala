// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package symmetric

import (
	"context"

	"github.com/absmach/supermq/auth"
	smqjwt "github.com/absmach/supermq/auth/tokenizer/util"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const patPrefix = "pat"

var errJWTExpiryKey = errors.New(`"exp" not satisfied`)

type tokenizer struct {
	algorithm jwa.KeyAlgorithm
	secret    []byte
	repo      auth.TokensRepository
	cache     auth.TokensCache
}

var _ auth.Tokenizer = (*tokenizer)(nil)

func NewTokenizer(algorithm string, secret []byte, repo auth.TokensRepository, cache auth.TokensCache) (auth.Tokenizer, error) {
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
		repo:      repo,
		cache:     cache,
	}, nil
}

func (tok *tokenizer) Issue(key auth.Key) (string, error) {
	tkn, err := smqjwt.BuildToken(key)
	if err != nil {
		return "", err
	}

	signedBytes, err := jwt.Sign(tkn, jwt.WithKey(tok.algorithm, tok.secret))
	if err != nil {
		return "", err
	}

	return string(signedBytes), nil
}

func (tok *tokenizer) Parse(ctx context.Context, tokenString string) (auth.Key, error) {
	key, err := tok.parseToken(tokenString)
	if err != nil {
		return auth.Key{}, err
	}
	if key.Type == auth.RefreshKey {
		switch tok.cache.Contains(ctx, key.ID) {
		case true:
			return auth.Key{}, auth.ErrRevokedToken
		default:
			if ok := tok.repo.Contains(ctx, key.ID); ok {
				if err := tok.cache.Save(ctx, key.ID); err != nil {
					return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
				}

				return auth.Key{}, auth.ErrRevokedToken
			}
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
		if err := tok.repo.Save(ctx, key.ID); err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}

		if err := tok.cache.Save(ctx, key.ID); err != nil {
			return errors.Wrap(svcerr.ErrAuthentication, err)
		}
	}

	return nil
}

func (tok *tokenizer) parseToken(tokenString string) (auth.Key, error) {
	if len(tokenString) >= 3 && tokenString[:3] == patPrefix {
		return auth.Key{Type: auth.PersonalAccessToken}, nil
	}

	tkn, err := jwt.Parse(
		[]byte(tokenString),
		jwt.WithValidate(true),
		jwt.WithKey(tok.algorithm, tok.secret),
	)
	if err != nil {
		if errors.Contains(err, errJWTExpiryKey) {
			return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, auth.ErrExpiry)
		}
		return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	if tkn.Issuer() != smqjwt.IssuerName {
		return auth.Key{}, smqjwt.ErrInvalidIssuer
	}

	return smqjwt.ToKey(tkn)
}
