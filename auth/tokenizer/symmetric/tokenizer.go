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

type tokenizer struct {
	algorithm jwa.KeyAlgorithm
	secret    []byte
}

var _ auth.Tokenizer = (*tokenizer)(nil)

func NewTokenizer(algorithm string, secret []byte) (auth.Tokenizer, error) {
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

func (tok *tokenizer) RetrieveJWKS() ([]auth.PublicKeyInfo, error) {
	return nil, auth.ErrPublicKeysNotSupported
}
