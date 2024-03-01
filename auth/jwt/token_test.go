// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package jwt_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	authjwt "github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const (
	tokenType   = "type"
	userField   = "user"
	domainField = "domain"
	issuerName  = "magistrala.auth"
	secret      = "test"
)

var (
	errInvalidIssuer = errors.New("invalid token issuer value")
	reposecret       = []byte("test")
)

func newToken(issuerName string, key auth.Key) string {
	builder := jwt.NewBuilder()
	builder.
		Issuer(issuerName).
		IssuedAt(key.IssuedAt).
		Subject(key.Subject).
		Claim(tokenType, "r").
		Expiration(key.ExpiresAt)
	builder.Claim(userField, key.User)
	builder.Claim(domainField, key.Domain)
	if key.ID != "" {
		builder.JwtID(key.ID)
	}
	tkn, _ := builder.Build()
	tokn, _ := jwt.Sign(tkn, jwt.WithKey(jwa.HS512, reposecret))
	return string(tokn)
}

func TestIssue(t *testing.T) {
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	tokenizer := authjwt.New([]byte(secret), provider)

	cases := []struct {
		desc string
		key  auth.Key
		err  error
	}{
		{
			desc: "issue new token",
			key:  key(),
			err:  nil,
		},
		{
			desc: "issue token with OAuth token",
			key: auth.Key{
				ID:        testsutil.GenerateUUID(t),
				Type:      auth.AccessKey,
				Subject:   testsutil.GenerateUUID(t),
				User:      testsutil.GenerateUUID(t),
				Domain:    testsutil.GenerateUUID(t),
				IssuedAt:  time.Now().Add(-10 * time.Second).Round(time.Second),
				ExpiresAt: time.Now().Add(10 * time.Minute).Round(time.Second),
				OAuth: auth.OAuthToken{
					Provider:     "test",
					AccessToken:  strings.Repeat("a", 1024),
					RefreshToken: strings.Repeat("b", 1024),
				},
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		tkn, err := tokenizer.Issue(tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err != nil {
			assert.NotEmpty(t, tkn, fmt.Sprintf("%s expected token, got empty string", tc.desc))
		}
	}
}

func TestParse(t *testing.T) {
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	tokenizer := authjwt.New([]byte(secret), provider)

	token, err := tokenizer.Issue(key())
	require.Nil(t, err, fmt.Sprintf("issuing key expected to succeed: %s", err))

	apiKey := key()
	apiKey.Type = auth.APIKey
	apiKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	apiToken, err := tokenizer.Issue(apiKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	expKey := key()
	expKey.ExpiresAt = time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expToken, err := tokenizer.Issue(expKey)
	require.Nil(t, err, fmt.Sprintf("issuing expired key expected to succeed: %s", err))

	inValidToken := newToken("invalid", key())

	cases := []struct {
		desc  string
		key   auth.Key
		token string
		err   error
	}{
		{
			desc:  "parse valid key",
			key:   key(),
			token: token,
			err:   nil,
		},
		{
			desc:  "parse invalid key",
			key:   auth.Key{},
			token: "invalid",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "parse expired key",
			key:   auth.Key{},
			token: expToken,
			err:   authjwt.ErrExpiry,
		},
		{
			desc:  "parse expired API key",
			key:   apiKey,
			token: apiToken,
			err:   authjwt.ErrExpiry,
		},
		{
			desc:  "parse token with invalid issuer",
			key:   auth.Key{},
			token: inValidToken,
			err:   errInvalidIssuer,
		},
		{
			desc:  "parse token with invalid content",
			key:   auth.Key{},
			token: newToken(issuerName, key()),
			err:   authjwt.ErrJSONHandle,
		},
	}

	for _, tc := range cases {
		key, err := tokenizer.Parse(tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.key, key, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.key, key))
		}
	}
}

func TestParseOAuthToken(t *testing.T) {
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	tokenizer := authjwt.New([]byte(secret), provider)

	validKey := oauthKey(t)
	invalidKey := oauthKey(t)
	invalidKey.OAuth.Provider = "invalid"

	cases := []struct {
		desc         string
		token        auth.Key
		issuedToken  string
		key          auth.Key
		validateErr  error
		refreshToken oauth2.Token
		refreshErr   error
		err          error
	}{
		{
			desc:        "parse valid key",
			token:       validKey,
			issuedToken: "",
			key:         validKey,
			validateErr: nil,
			refreshErr:  nil,
			err:         nil,
		},
		{
			desc:        "parse invalid key but refreshed",
			token:       validKey,
			issuedToken: "",
			key:         validKey,
			validateErr: svcerr.ErrAuthentication,
			refreshToken: oauth2.Token{
				AccessToken:  strings.Repeat("a", 10),
				RefreshToken: strings.Repeat("b", 10),
			},
			refreshErr: nil,
			err:        nil,
		},
		{
			desc:         "parse invalid key but not refreshed",
			token:        validKey,
			issuedToken:  "",
			key:          validKey,
			validateErr:  svcerr.ErrAuthentication,
			refreshToken: oauth2.Token{},
			refreshErr:   svcerr.ErrAuthentication,
			err:          svcerr.ErrAuthentication,
		},
		{
			desc:        "parse invalid key with different provider",
			issuedToken: invalidOauthToken(t, invalidKey, "invalid", "a", "b"),
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "parse invalid key with invalid access token",
			issuedToken: invalidOauthToken(t, invalidKey, "invalid", 123, "b"),
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "parse invalid key with invalid refresh token",
			issuedToken: invalidOauthToken(t, invalidKey, "invalid", "a", 123),
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "parse invalid key with invalid provider",
			issuedToken: invalidOauthToken(t, invalidKey, "test", "a", "b"),
			err:         svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		tokenCall := provider.On("Name").Return("test")
		tokenCall1 := provider.On("Validate", context.Background(), mock.Anything).Return(tc.validateErr)
		tokenCall2 := provider.On("Refresh", context.Background(), mock.Anything).Return(tc.refreshToken, tc.refreshErr)
		if tc.issuedToken == "" {
			var err error
			tc.issuedToken, err = tokenizer.Issue(tc.token)
			require.Nil(t, err, fmt.Sprintf("issuing key expected to succeed: %s", err))
		}
		key, err := tokenizer.Parse(tc.issuedToken)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.key, key, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.key, key))
		}
		tokenCall.Unset()
		tokenCall1.Unset()
		tokenCall2.Unset()
	}
}

func key() auth.Key {
	exp := time.Now().UTC().Add(10 * time.Minute).Round(time.Second)
	return auth.Key{
		ID:        "66af4a67-3823-438a-abd7-efdb613eaef6",
		Type:      auth.AccessKey,
		Issuer:    "magistrala.auth",
		Subject:   "66af4a67-3823-438a-abd7-efdb613eaef6",
		IssuedAt:  time.Now().UTC().Add(-10 * time.Second).Round(time.Second),
		ExpiresAt: exp,
	}
}

func oauthKey(t *testing.T) auth.Key {
	return auth.Key{
		ID:        testsutil.GenerateUUID(t),
		Type:      auth.AccessKey,
		Issuer:    "magistrala.auth",
		Subject:   testsutil.GenerateUUID(t),
		User:      testsutil.GenerateUUID(t),
		Domain:    testsutil.GenerateUUID(t),
		IssuedAt:  time.Now().UTC().Add(-10 * time.Second).Round(time.Second),
		ExpiresAt: time.Now().UTC().Add(10 * time.Minute).Round(time.Second),
		OAuth: auth.OAuthToken{
			Provider:     "test",
			AccessToken:  strings.Repeat("a", 10),
			RefreshToken: strings.Repeat("b", 10),
		},
	}
}

func invalidOauthToken(t *testing.T, key auth.Key, provider, accessToken, refreshToken interface{}) string {
	builder := jwt.NewBuilder()
	builder.
		Issuer(issuerName).
		IssuedAt(key.IssuedAt).
		Subject(key.Subject).
		Claim(tokenType, key.Type).
		Expiration(key.ExpiresAt)
	builder.Claim(userField, key.User)
	builder.Claim(domainField, key.Domain)
	if provider != nil {
		builder.Claim("oauth_provider", provider)
		if accessToken != nil {
			builder.Claim(provider.(string), map[string]interface{}{"access_token": accessToken})
		}
		if refreshToken != nil {
			builder.Claim(provider.(string), map[string]interface{}{"refresh_token": refreshToken})
		}
	}
	if key.ID != "" {
		builder.JwtID(key.ID)
	}
	tkn, err := builder.Build()
	require.Nil(t, err, fmt.Sprintf("building token expected to succeed: %s", err))
	signedTkn, err := jwt.Sign(tkn, jwt.WithKey(jwa.HS512, reposecret))
	require.Nil(t, err, fmt.Sprintf("signing token expected to succeed: %s", err))
	return string(signedTkn)
}
