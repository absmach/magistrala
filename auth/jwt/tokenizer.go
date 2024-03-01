// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/oauth2"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

var (
	errInvalidIssuer = errors.New("invalid token issuer value")
	// errJWTExpiryKey is used to check if the token is expired.
	errJWTExpiryKey = errors.New(`"exp" not satisfied`)
	// ErrExpiry indicates that the token is expired.
	ErrExpiry = errors.New("token is expired")
	// ErrSetClaim indicates an inability to set the claim.
	ErrSetClaim = errors.New("failed to set claim")
	// ErrSignJWT indicates an error in signing jwt token.
	ErrSignJWT = errors.New("failed to sign jwt token")
	// ErrParseToken indicates a failure to parse the token.
	ErrParseToken = errors.New("failed to parse token")
	// ErrValidateJWTToken indicates a failure to validate JWT token.
	ErrValidateJWTToken = errors.New("failed to validate jwt token")
	// ErrJSONHandle indicates an error in handling JSON.
	ErrJSONHandle = errors.New("failed to perform operation JSON")

	// errInvalidProvider indicates an invalid OAuth2.0 provider.
	errInvalidProvider = errors.New("invalid OAuth2.0 provider")
)

const (
	issuerName             = "magistrala.auth"
	tokenType              = "type"
	userField              = "user"
	domainField            = "domain"
	oauthProviderField     = "oauth_provider"
	oauthAccessTokenField  = "access_token"
	oauthRefreshTokenField = "refresh_token"
)

type tokenizer struct {
	secret    []byte
	providers map[string]oauth2.Provider
}

var _ auth.Tokenizer = (*tokenizer)(nil)

// NewRepository instantiates an implementation of Token repository.
func New(secret []byte, providers ...oauth2.Provider) auth.Tokenizer {
	providersMap := make(map[string]oauth2.Provider)
	for _, provider := range providers {
		providersMap[provider.Name()] = provider
	}
	return &tokenizer{
		secret:    secret,
		providers: providersMap,
	}
}

func (tok *tokenizer) Issue(key auth.Key) (string, error) {
	builder := jwt.NewBuilder()
	builder.
		Issuer(issuerName).
		IssuedAt(key.IssuedAt).
		Subject(key.Subject).
		Claim(tokenType, key.Type).
		Expiration(key.ExpiresAt)
	builder.Claim(userField, key.User)
	builder.Claim(domainField, key.Domain)

	if key.OAuth.Provider != "" {
		provider, ok := tok.providers[key.OAuth.Provider]
		if !ok {
			return "", errors.Wrap(svcerr.ErrAuthentication, errInvalidProvider)
		}
		builder.Claim(oauthProviderField, provider.Name())
		builder.Claim(provider.Name(), map[string]interface{}{
			oauthAccessTokenField:  key.OAuth.AccessToken,
			oauthRefreshTokenField: key.OAuth.RefreshToken,
		})
	}

	if key.ID != "" {
		builder.JwtID(key.ID)
	}
	tkn, err := builder.Build()
	if err != nil {
		return "", errors.Wrap(svcerr.ErrAuthentication, err)
	}
	signedTkn, err := jwt.Sign(tkn, jwt.WithKey(jwa.HS512, tok.secret))
	if err != nil {
		return "", errors.Wrap(ErrSignJWT, err)
	}
	return string(signedTkn), nil
}

func (tok *tokenizer) Parse(token string) (auth.Key, error) {
	tkn, err := jwt.Parse(
		[]byte(token),
		jwt.WithValidate(true),
		jwt.WithKey(jwa.HS512, tok.secret),
	)
	if err != nil {
		if errors.Contains(err, errJWTExpiryKey) {
			return auth.Key{}, ErrExpiry
		}

		return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	validator := jwt.ValidatorFunc(func(_ context.Context, t jwt.Token) jwt.ValidationError {
		if t.Issuer() != issuerName {
			return jwt.NewValidationError(errInvalidIssuer)
		}
		return nil
	})
	if err := jwt.Validate(tkn, jwt.WithValidator(validator)); err != nil {
		return auth.Key{}, errors.Wrap(ErrValidateJWTToken, err)
	}

	jsn, err := json.Marshal(tkn.PrivateClaims())
	if err != nil {
		return auth.Key{}, errors.Wrap(ErrJSONHandle, err)
	}
	var key auth.Key
	if err := json.Unmarshal(jsn, &key); err != nil {
		return auth.Key{}, errors.Wrap(ErrJSONHandle, err)
	}

	tType, ok := tkn.Get(tokenType)
	if !ok {
		return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}
	ktype, err := strconv.ParseInt(fmt.Sprintf("%v", tType), 10, 64)
	if err != nil {
		return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
	}

	key.ID = tkn.JwtID()
	key.Type = auth.KeyType(ktype)
	key.Issuer = tkn.Issuer()
	key.Subject = tkn.Subject()
	key.IssuedAt = tkn.IssuedAt()
	key.ExpiresAt = tkn.Expiration()

	oauthProvider, ok := tkn.Get(oauthProviderField)
	if ok {
		provider, ok := oauthProvider.(string)
		if !ok {
			return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, errInvalidProvider)
		}
		if provider != "" {
			prov, ok := tok.providers[provider]
			if !ok {
				return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, errInvalidProvider)
			}
			key.OAuth.Provider = prov.Name()

			key, err = parseOAuthToken(context.Background(), prov, tkn, key)
			if err != nil {
				return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
			}

			return key, nil
		}
	}

	return key, nil
}

func parseOAuthToken(ctx context.Context, provider oauth2.Provider, token jwt.Token, key auth.Key) (auth.Key, error) {
	oauthToken, ok := token.Get(provider.Name())
	if ok {
		claims, ok := oauthToken.(map[string]interface{})
		if !ok {
			return auth.Key{}, errors.Wrap(ErrParseToken, fmt.Errorf("invalid claims for %s token", provider.Name()))
		}
		accessToken, ok := claims[oauthAccessTokenField].(string)
		if !ok {
			return auth.Key{}, errors.Wrap(ErrParseToken, fmt.Errorf("invalid access token claim for %s token", provider.Name()))
		}
		refreshToken, ok := claims[oauthRefreshTokenField].(string)
		if !ok {
			return auth.Key{}, errors.Wrap(ErrParseToken, fmt.Errorf("invalid refresh token claim for %s token", provider.Name()))
		}

		switch provider.Validate(ctx, accessToken) {
		case nil:
			key.OAuth.AccessToken = accessToken
		default:
			token, err := provider.Refresh(ctx, refreshToken)
			if err != nil {
				return auth.Key{}, errors.Wrap(svcerr.ErrAuthentication, err)
			}
			key.OAuth.AccessToken = token.AccessToken
			key.OAuth.RefreshToken = token.RefreshToken

			return key, nil
		}

		key.OAuth.RefreshToken = refreshToken

		return key, nil
	}

	return key, nil
}
