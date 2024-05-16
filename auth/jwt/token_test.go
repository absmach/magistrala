// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package jwt_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	authjwt "github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		Claim(tokenType, "r").
		Expiration(key.ExpiresAt)
	builder.Claim(userField, key.User)
	if key.Domain != "" {
		builder.Claim(domainField, key.Domain)
	}
	if key.Subject != "" {
		builder.Subject(key.Subject)
	}
	if key.ID != "" {
		builder.JwtID(key.ID)
	}
	tkn, _ := builder.Build()
	tokn, _ := jwt.Sign(tkn, jwt.WithKey(jwa.HS512, reposecret))
	return string(tokn)
}

func TestIssue(t *testing.T) {
	repo := new(mocks.TokenRepository)
	cache := new(mocks.Cache)
	tokenizer := authjwt.New([]byte(secret), repo, cache)

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
			},
			err: nil,
		},
		{
			desc: "issue token without a domain",
			key: auth.Key{
				ID:       testsutil.GenerateUUID(t),
				Type:     auth.AccessKey,
				Subject:  testsutil.GenerateUUID(t),
				User:     testsutil.GenerateUUID(t),
				Domain:   "",
				IssuedAt: time.Now().Add(-10 * time.Second).Round(time.Second),
			},
			err: nil,
		},
		{
			desc: "issue token without a subject",
			key: auth.Key{
				ID:       testsutil.GenerateUUID(t),
				Type:     auth.AccessKey,
				Subject:  "",
				User:     testsutil.GenerateUUID(t),
				Domain:   testsutil.GenerateUUID(t),
				IssuedAt: time.Now().Add(-10 * time.Second).Round(time.Second),
			},
			err: nil,
		},
		{
			desc: "issue token without a domain and subject",
			key: auth.Key{
				ID:        testsutil.GenerateUUID(t),
				Type:      auth.AccessKey,
				Subject:   "",
				User:      testsutil.GenerateUUID(t),
				Domain:    "",
				IssuedAt:  time.Now().Add(-10 * time.Second).Round(time.Second),
				ExpiresAt: time.Now().Add(10 * time.Minute).Round(time.Second),
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
	repo := new(mocks.TokenRepository)
	cache := new(mocks.Cache)
	tokenizer := authjwt.New([]byte(secret), repo, cache)

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

	emptyDomainKey := key()
	emptyDomainKey.Domain = ""
	emptyDomainToken, err := tokenizer.Issue(emptyDomainKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	emptySubjectKey := key()
	emptySubjectKey.Subject = ""
	emptySubjectToken, err := tokenizer.Issue(emptySubjectKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	emptyKey := key()
	emptyKey.Domain = ""
	emptyKey.Subject = ""
	emptyToken, err := tokenizer.Issue(emptyKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	inValidToken := newToken("invalid", key())

	refreshKey := key()
	refreshKey.Type = auth.RefreshKey
	refreshToken, err := tokenizer.Issue(refreshKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc          string
		key           auth.Key
		token         string
		cacheContains bool
		repoContains  bool
		cacheSave     error
		err           error
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
			err:   auth.ErrExpiry,
		},
		{
			desc:  "parse expired API key",
			key:   apiKey,
			token: apiToken,
			err:   auth.ErrExpiry,
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
		{
			desc:  "parse token with empty domain",
			key:   emptyDomainKey,
			token: emptyDomainToken,
			err:   nil,
		},
		{
			desc:  "parse token with empty subject",
			key:   emptySubjectKey,
			token: emptySubjectToken,
			err:   nil,
		},
		{
			desc:  "parse token with empty domain and subject",
			key:   emptyKey,
			token: emptyToken,
			err:   nil,
		},
		{
			desc:          "parse refresh token",
			key:           refreshKey,
			token:         refreshToken,
			cacheContains: false,
			repoContains:  false,
			err:           nil,
		},
		{
			desc:          "parse revoked refresh token in cache",
			key:           refreshKey,
			token:         refreshToken,
			cacheContains: true,
			repoContains:  false,
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "parse revoked refresh token not in cache",
			key:           refreshKey,
			token:         refreshToken,
			cacheContains: false,
			repoContains:  true,
			err:           svcerr.ErrAuthentication,
		},
		{
			desc:          "parse revoked refresh token failed to save in cache",
			key:           refreshKey,
			token:         refreshToken,
			cacheContains: false,
			repoContains:  true,
			cacheSave:     repoerr.ErrCreateEntity,
			err:           svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		cacheCall := cache.On("Contains", context.Background(), "", tc.key.ID).Return(tc.cacheContains)
		repoCall := repo.On("Contains", context.Background(), tc.key.ID).Return(tc.repoContains)
		cacheCall1 := cache.On("Save", context.Background(), "", tc.key.ID).Return(tc.cacheSave)
		key, err := tokenizer.Parse(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, tc.key, key, fmt.Sprintf("%s expected %v, got %v", tc.desc, tc.key, key))
		}
		cacheCall.Unset()
		repoCall.Unset()
		cacheCall1.Unset()
	}
}

func TestRevoke(t *testing.T) {
	repo := new(mocks.TokenRepository)
	cache := new(mocks.Cache)
	tokenizer := authjwt.New([]byte(secret), repo, cache)

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

	emptyDomainKey := key()
	emptyDomainKey.Domain = ""
	emptyDomainToken, err := tokenizer.Issue(emptyDomainKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	emptySubjectKey := key()
	emptySubjectKey.Subject = ""
	emptySubjectToken, err := tokenizer.Issue(emptySubjectKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	emptyKey := key()
	emptyKey.Domain = ""
	emptyKey.Subject = ""
	emptyToken, err := tokenizer.Issue(emptyKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	inValidToken := newToken("invalid", key())

	refreshKey := key()
	refreshKey.Type = auth.RefreshKey
	refreshToken, err := tokenizer.Issue(refreshKey)
	require.Nil(t, err, fmt.Sprintf("issuing user key expected to succeed: %s", err))

	cases := []struct {
		desc     string
		key      auth.Key
		token    string
		repoErr  error
		cacheErr error
		err      error
	}{
		{
			desc:  "revoke valid key",
			key:   key(),
			token: token,
			err:   nil,
		},
		{
			desc:  "revoke invalid key",
			key:   auth.Key{},
			token: "invalid",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "revoke expired key",
			key:   auth.Key{},
			token: expToken,
			err:   auth.ErrExpiry,
		},
		{
			desc:  "revoke expired API key",
			key:   apiKey,
			token: apiToken,
			err:   auth.ErrExpiry,
		},
		{
			desc:  "revoke token with invalid issuer",
			key:   auth.Key{},
			token: inValidToken,
			err:   errInvalidIssuer,
		},
		{
			desc:  "revoke token with invalid content",
			key:   auth.Key{},
			token: newToken(issuerName, key()),
			err:   authjwt.ErrJSONHandle,
		},
		{
			desc:  "revoke token with empty domain",
			key:   emptyDomainKey,
			token: emptyDomainToken,
			err:   nil,
		},
		{
			desc:  "revoke token with empty subject",
			key:   emptySubjectKey,
			token: emptySubjectToken,
			err:   nil,
		},
		{
			desc:  "revoke token with empty domain and subject",
			key:   emptyKey,
			token: emptyToken,
			err:   nil,
		},
		{
			desc:  "revoke refresh token",
			key:   refreshKey,
			token: refreshToken,
			err:   nil,
		},
		{
			desc:     "revoke revoked refresh token failed to save in cache",
			key:      refreshKey,
			token:    refreshToken,
			repoErr:  nil,
			cacheErr: repoerr.ErrCreateEntity,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "revoke revoked refresh token failed to save in cache",
			key:      refreshKey,
			token:    refreshToken,
			repoErr:  repoerr.ErrCreateEntity,
			cacheErr: nil,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repoCall := repo.On("Save", context.Background(), tc.key.ID).Return(tc.repoErr)
		cacheCall := cache.On("Save", context.Background(), "", tc.key.ID).Return(tc.cacheErr)
		err := tokenizer.Revoke(context.Background(), tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s, got %s", tc.desc, tc.err, err))
		cacheCall.Unset()
		repoCall.Unset()
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
