// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/supermq/auth"
	"github.com/absmach/supermq/auth/mocks"
	"github.com/absmach/supermq/internal/testsutil"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	policymocks "github.com/absmach/supermq/pkg/policies/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	tokenType       = "type"
	roleField       = "role"
	VerifiedField   = "verified"
	issuerName      = "supermq.auth"
)

var (
	errRoleAuth  = errors.New("failed to authorize user role")
	ErrExpiry    = errors.New("token is expired")
	inValidToken = "invalid"
	userID       = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
	accessKey    = auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.AccessKey,
		Role:      auth.UserRole,
		Issuer:    issuerName,
	}
)

var (
	krepo      *mocks.KeyRepository
	pService   *policymocks.Service
	pEvaluator *policymocks.Evaluator
	patsrepo   *mocks.PATSRepository
	cache      *mocks.Cache
	hasher     *mocks.Hasher
	tokenizer  *mocks.Tokenizer
)

func newService(t *testing.T) (auth.Service, string) {
	krepo = new(mocks.KeyRepository)
	cache = new(mocks.Cache)
	pService = new(policymocks.Service)
	pEvaluator = new(policymocks.Evaluator)
	patsrepo = new(mocks.PATSRepository)
	hasher = new(mocks.Hasher)
	idProvider := uuid.NewMock()
	tokenizer = new(mocks.Tokenizer)

	token, _, err := signToken(t, issuerName, accessKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing access key expected to succeed: %s", err))

	return auth.New(krepo, patsrepo, cache, hasher, idProvider, tokenizer, pEvaluator, pService, loginDuration, refreshDuration, invalidDuration), token
}

func TestIssue(t *testing.T) {
	svc, accessToken := newService(t)

	accesskey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.AccessKey,
		Role:      auth.UserRole,
		Issuer:    issuerName,
	}
	apikey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.APIKey,
		Role:      auth.UserRole,
	}
	apiToken, _, err := signToken(t, issuerName, apikey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	refreshkey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.RefreshKey,
		Role:      auth.UserRole,
	}
	refreshToken, _, err := signToken(t, issuerName, refreshkey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing refresh key expected to succeed: %s", err))

	cases := []struct {
		desc         string
		key          auth.Key
		token        string
		roleCheckErr error
		tokenizerErr error
		err          error
	}{
		{
			desc: "issue recovery key",
			key: auth.Key{
				Type:     auth.RecoveryKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Issue", mock.Anything).Return(tc.token, tc.tokenizerErr)
			policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
				Subject:     tc.key.Subject,
				SubjectType: policies.UserType,
				Permission:  policies.MembershipPermission,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
			}).Return(tc.roleCheckErr)
			_, err := svc.Issue(context.Background(), tc.token, tc.key)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			policyCall.Unset()
			tokenizerCall.Unset()
		})
	}

	cases2 := []struct {
		desc         string
		key          auth.Key
		saveResponse auth.Key
		token        string
		tokenizerErr error
		saveErr      error
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue access key",
			key: auth.Key{
				Type:     auth.AccessKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token: accessToken,
			err:   nil,
		},
	}
	for _, tc := range cases2 {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Issue", mock.Anything).Return(tc.token, tc.tokenizerErr)
			repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
			policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
				Subject:     tc.key.Subject,
				SubjectType: policies.UserType,
				Permission:  policies.MembershipPermission,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
			}).Return(tc.roleCheckErr)
			_, err := svc.Issue(context.Background(), tc.token, tc.key)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			tokenizerCall.Unset()
			repoCall.Unset()
			policyCall.Unset()
		})
	}

	cases3 := []struct {
		desc         string
		key          auth.Key
		token        string
		issueErr     error
		parseRes     auth.Key
		parseErr     error
		saveErr      error
		roleCheckErr error
		err          error
	}{
		{
			desc: "issue API key",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:    accessToken,
			parseRes: accesskey,
			err:      nil,
		},
		{
			desc: "issue API key with an invalid token",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:    "invalid",
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: " issue API key with invalid key request",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  "",
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:    apiToken,
			issueErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "issue API key with failed to save",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:    accessToken,
			parseRes: accesskey,
			saveErr:  repoerr.ErrNotFound,
			err:      repoerr.ErrNotFound,
		},
		{
			desc: "issue API key with failed to check role",
			key: auth.Key{
				Type:     auth.APIKey,
				Subject:  userID,
				Role:     auth.UserRole,
				IssuedAt: time.Now(),
			},
			token:        accessToken,
			parseRes:     accesskey,
			roleCheckErr: errRoleAuth,
			err:          errRoleAuth,
		},
	}
	for _, tc := range cases3 {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Issue", mock.Anything).Return(tc.token, tc.issueErr)
			tokenizerCall1 := tokenizer.On("Parse", mock.Anything, tc.token).Return(tc.parseRes, tc.parseErr)
			repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
			policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
				Subject:     tc.key.Subject,
				SubjectType: policies.UserType,
				Permission:  policies.MembershipPermission,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
			}).Return(tc.roleCheckErr)
			_, err := svc.Issue(context.Background(), tc.token, tc.key)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			tokenizerCall.Unset()
			tokenizerCall1.Unset()
			repoCall.Unset()
			policyCall.Unset()
		})
	}

	cases4 := []struct {
		desc         string
		key          auth.Key
		token        string
		parseRes     auth.Key
		parseErr     error
		roleCheckErr error
		issueErr     error
		err          error
	}{
		{
			desc: "issue refresh key",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token:    refreshToken,
			parseRes: refreshkey,
			err:      nil,
		},
		{
			desc: "issue refresh key with invalid token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token:    inValidToken,
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "issue refresh key with empty token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token:    "",
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc: "issue refresh key with failed to check role",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
				Subject:  userID,
				Role:     auth.UserRole,
			},
			token:        refreshToken,
			parseRes:     refreshkey,
			roleCheckErr: errRoleAuth,
			err:          errRoleAuth,
		},
	}
	for _, tc := range cases4 {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Issue", mock.Anything).Return(tc.token, tc.issueErr)
			tokenizerCall1 := tokenizer.On("Parse", mock.Anything, tc.token).Return(tc.parseRes, tc.parseErr)
			policyCall := pEvaluator.On("CheckPolicy", mock.Anything, policies.Policy{
				Subject:     tc.key.Subject,
				SubjectType: policies.UserType,
				Permission:  policies.MembershipPermission,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
			}).Return(tc.roleCheckErr)
			_, err := svc.Issue(context.Background(), tc.token, tc.key)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			tokenizerCall.Unset()
			tokenizerCall1.Unset()
			policyCall.Unset()
		})
	}
}

func TestRevoke(t *testing.T) {
	svc, _ := newService(t)

	accesskey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   userID,
		Type:      auth.AccessKey,
		Role:      auth.UserRole,
		Issuer:    issuerName,
	}
	apikey := auth.Key{
		Type:     auth.APIKey,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
		Subject:  userID,
	}
	apiToken, _, err := signToken(t, issuerName, apikey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	cases := []struct {
		desc     string
		id       string
		token    string
		parseRes auth.Key
		parseErr error
		err      error
	}{
		{
			desc:     "revoke login key",
			token:    apiToken,
			parseRes: accesskey,
			err:      nil,
		},
		{
			desc:     "revoke non-existing login key",
			token:    apiToken,
			parseRes: accesskey,
			err:      nil,
		},
		{
			desc:     "revoke with empty login key",
			token:    "",
			parseRes: auth.Key{},
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "revoke login key with failed to remove",
			id:       "invalidID",
			token:    apiToken,
			parseRes: accesskey,
			err:      svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Parse", mock.Anything, tc.token).Return(tc.parseRes, tc.parseErr)
			repoCall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			err := svc.Revoke(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			tokenizerCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestRetrieve(t *testing.T) {
	svc, accessToken := newService(t)

	apiKey := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  userID,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
	}

	apiToken, _, err := signToken(t, issuerName, apiKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	recoveryKey := auth.Key{
		Type:     auth.RecoveryKey,
		Subject:  userID,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
	}
	resetToken, _, err := signToken(t, issuerName, recoveryKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	cases := []struct {
		desc     string
		id       string
		token    string
		parseRes auth.Key
		parseErr error
		err      error
	}{
		{
			desc:     "retrieve login key",
			token:    accessToken,
			parseRes: accessKey,
			err:      nil,
		},
		{
			desc:     "retrieve non-existing login key",
			id:       "invalid",
			token:    accessToken,
			parseRes: accessKey,
			err:      svcerr.ErrNotFound,
		},
		{
			desc:     "retrieve with wrong login key",
			token:    "wrong",
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "retrieve with API token",
			token:    apiToken,
			parseRes: apiKey,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "retrieve with reset token",
			token:    resetToken,
			parseRes: recoveryKey,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Parse", mock.Anything, tc.token).Return(tc.parseRes, tc.parseErr)
			repoCall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
			_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			tokenizerCall.Unset()
			repoCall.Unset()
		})
	}
}

func TestIdentify(t *testing.T) {
	svc, accessToken := newService(t)

	refreshKey := auth.Key{
		Type:     auth.RefreshKey,
		Role:     auth.UserRole,
		Subject:  userID,
		IssuedAt: time.Now(),
	}
	refreshToken, _, err := signToken(t, issuerName, refreshKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing refresh key expected to succeed: %s", err))

	recoveryKey := auth.Key{
		Type:     auth.RecoveryKey,
		Role:     auth.UserRole,
		IssuedAt: time.Now(),
		Subject:  userID,
	}
	recoverySecret, _, err := signToken(t, issuerName, recoveryKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing recovery key expected to succeed: %s", err))

	apiKey := auth.Key{
		Type:      auth.APIKey,
		Role:      auth.UserRole,
		Subject:   userID,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Minute),
	}
	apiSecret, _, err := signToken(t, issuerName, apiKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	exp0 := time.Now().UTC().Add(-10 * time.Second).Round(time.Second)
	exp1 := time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expiredKey := auth.Key{
		Type:      auth.APIKey,
		Role:      auth.UserRole,
		Subject:   userID,
		IssuedAt:  exp0,
		ExpiresAt: exp1,
	}
	expSecret, _, err := signToken(t, issuerName, expiredKey, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing expired API key expected to succeed: %s", err))

	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Type:      7,
		Subject:   userID,
		Role:      auth.UserRole,
	}
	invalidTokenType, _, err := signToken(t, issuerName, key, false)
	assert.Nil(t, err, fmt.Sprintf("Issuing invalid token type key expected to succeed: %s", err))

	cases := []struct {
		desc     string
		key      string
		subject  string
		parseRes auth.Key
		parseErr error
		err      error
	}{
		{
			desc:     "identify login key",
			key:      accessToken,
			subject:  userID,
			parseRes: accessKey,
			err:      nil,
		},
		{
			desc:     "identify refresh key",
			key:      refreshToken,
			subject:  userID,
			parseRes: refreshKey,
			err:      nil,
		},
		{
			desc:     "identify recovery key",
			key:      recoverySecret,
			subject:  userID,
			parseRes: recoveryKey,
			err:      nil,
		},
		{
			desc:     "identify API key",
			key:      apiSecret,
			subject:  userID,
			parseRes: apiKey,
			err:      nil,
		},
		{
			desc:     "identify expired API key",
			key:      expSecret,
			subject:  "",
			parseErr: ErrExpiry,
			err:      auth.ErrKeyExpired,
		},
		{
			desc:     "identify API key with failed to retrieve",
			key:      apiSecret,
			subject:  "",
			parseRes: apiKey,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "identify invalid key",
			key:      "invalid",
			subject:  "",
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "identify invalid key type",
			key:      invalidTokenType,
			subject:  "",
			parseErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			tokenizerCall := tokenizer.On("Parse", mock.Anything, tc.key).Return(tc.parseRes, tc.parseErr)
			repoCall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
			repoCall1 := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
			idt, err := svc.Identify(context.Background(), tc.key)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.subject, idt.Subject, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.subject, idt))
			tokenizerCall.Unset()
			repoCall.Unset()
			repoCall1.Unset()
		})
	}
}

func TestAuthorize(t *testing.T) {
	svc, _ := newService(t)

	cases := []struct {
		desc                 string
		policyReq            policies.Policy
		patAuthz             *auth.PATAuthz
		checkDomainPolicyReq policies.Policy
		checkPolicyReq       policies.Policy
		patErr               error
		parseReq             string
		parseRes             auth.Key
		parseErr             error
		callBackErr          error
		checkPolicyErr       error
		checkDomainPolicyErr error
		authorizePATErr      error
		err                  error
	}{

		{
			desc: "authorize a user key successfully",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      validID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			err: nil,
		},
		{
			desc: "authorize with PAT successfully",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     userID,
				Object:      validID,
				ObjectType:  policies.ClientType,
				Permission:  policies.ViewPermission,
				Domain:      domainID,
			},
			patAuthz: &auth.PATAuthz{
				PatID:      validID,
				UserID:     userID,
				EntityType: auth.ClientsType,
				EntityID:   validID,
				Operation:  "read",
				Domain:     domainID,
			},
			checkDomainPolicyReq: policies.Policy{
				Subject:     userID,
				SubjectType: policies.UserType,
				Object:      domainID,
				ObjectType:  policies.DomainType,
				Permission:  policies.MembershipPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     userID,
				Object:      validID,
				ObjectType:  policies.ClientType,
				Permission:  policies.ViewPermission,
				Domain:      domainID,
			},
			err: nil,
		},
		{
			desc: "authorize with PAT scope check failure",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			patAuthz: &auth.PATAuthz{
				PatID:      validID,
				UserID:     userID,
				EntityType: auth.ChannelsType,
				Domain:     domainID,
				Operation:  auth.OpListChannels,
				EntityID:   auth.AnyIDs,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			patErr: svcerr.ErrAuthorization,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc: "authorize with invalid PAT entity type",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			patAuthz: &auth.PATAuthz{
				PatID:      validID,
				UserID:     userID,
				EntityType: auth.EntityType(100),
				Domain:     domainID,
				Operation:  auth.OpListChannels,
				EntityID:   auth.AnyIDs,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			patErr: errors.New("unknown domain entity type invalid"),
			err:    errors.New("unknown domain entity type invalid"),
		},

		{
			desc: "authorize with PAT but PAT authorization fails",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     userID,
				Object:      validID,
				ObjectType:  policies.ClientType,
				Permission:  policies.ViewPermission,
			},
			patAuthz: &auth.PATAuthz{
				PatID:      validID,
				UserID:     userID,
				EntityType: auth.ClientsType,
				EntityID:   validID,
				Operation:  "read",
				Domain:     domainID,
			},
			checkPolicyReq:  policies.Policy{},
			authorizePATErr: svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},

		{
			desc: "authorize with PAT - PAT authorization fails but policy check not reached",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Subject:     userID,
				Object:      validID,
				ObjectType:  policies.ClientType,
				Permission:  policies.ViewPermission,
			},
			patAuthz: &auth.PATAuthz{
				PatID:      validID,
				UserID:     userID,
				EntityType: auth.ClientsType,
				EntityID:   validID,
				Operation:  "write",
				Domain:     domainID,
			},
			checkPolicyReq:  policies.Policy{},
			authorizePATErr: svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc: "authorize with policy check error",
			policyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyReq: policies.Policy{
				SubjectType: policies.UserType,
				SubjectKind: policies.UsersKind,
				Object:      policies.SuperMQObject,
				ObjectType:  policies.PlatformType,
				Permission:  policies.AdminPermission,
			},
			checkPolicyErr: repoerr.ErrNotFound,
			err:            svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var policyCall *mock.Call
			if tc.checkPolicyReq != (policies.Policy{}) {
				policyCall = pEvaluator.On("CheckPolicy", mock.Anything, tc.checkPolicyReq).Return(tc.checkPolicyErr)
			}
			var patCall *mock.Call
			if tc.patAuthz != nil {
				patErr := tc.patErr
				if patErr == nil {
					patErr = tc.authorizePATErr
				}
				patCall = patsrepo.On("CheckScope", mock.Anything, tc.patAuthz.UserID, tc.patAuthz.PatID, tc.patAuthz.EntityType, tc.patAuthz.Domain, tc.patAuthz.Operation, tc.patAuthz.EntityID).Return(patErr)
			}
			repoCall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			err := svc.Authorize(context.Background(), tc.policyReq, tc.patAuthz)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
			if policyCall != nil {
				policyCall.Unset()
			}
			if patCall != nil {
				patCall.Unset()
			}
			repoCall.Unset()
		})
	}
}

func TestSwitchToPermission(t *testing.T) {
	cases := []struct {
		desc     string
		relation string
		result   string
	}{
		{
			desc:     "switch to admin permission",
			relation: policies.AdministratorRelation,
			result:   policies.AdminPermission,
		},
		{
			desc:     "switch to editor permission",
			relation: policies.EditorRelation,
			result:   policies.EditPermission,
		},
		{
			desc:     "switch to contributor permission",
			relation: policies.ContributorRelation,
			result:   policies.ViewPermission,
		},
		{
			desc:     "switch to member permission",
			relation: policies.MemberRelation,
			result:   policies.MembershipPermission,
		},
		{
			desc:     "switch to group permission",
			relation: policies.GroupRelation,
			result:   policies.GroupRelation,
		},
		{
			desc:     "switch to guest permission",
			relation: policies.GuestRelation,
			result:   policies.ViewPermission,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			result := auth.SwitchToPermission(tc.relation)
			assert.Equal(t, tc.result, result, fmt.Sprintf("switching to permission expected to succeed: %s", result))
		})
	}
}

func TestEncodeDomainUserID(t *testing.T) {
	cases := []struct {
		desc     string
		domainID string
		userID   string
		response string
	}{
		{
			desc:     "encode domain user id successfully",
			domainID: validID,
			userID:   validID,
			response: validID + "_" + validID,
		},
		{
			desc:     "encode domain user id with empty userID",
			domainID: validID,
			userID:   "",
			response: "",
		},
		{
			desc:     "encode domain user id with empty domain ID",
			domainID: "",
			userID:   validID,
			response: "",
		},
		{
			desc:     "encode domain user id with empty domain ID and userID",
			domainID: "",
			userID:   "",
			response: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ar := auth.EncodeDomainUserID(tc.domainID, tc.userID)
			assert.Equal(t, tc.response, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.response, ar))
		})
	}
}

func TestDecodeDomainUserID(t *testing.T) {
	cases := []struct {
		desc         string
		domainUserID string
		respDomainID string
		respUserID   string
	}{
		{
			desc:         "decode domain user id successfully",
			domainUserID: validID + "_" + validID,
			respDomainID: validID,
			respUserID:   validID,
		},
		{
			desc:         "decode domain user id with empty domainUserID",
			domainUserID: "",
			respDomainID: "",
			respUserID:   "",
		},
		{
			desc:         "decode domain user id with empty UserID",
			domainUserID: validID,
			respDomainID: validID,
			respUserID:   "",
		},
		{
			desc:         "decode domain user id with invalid domainuserId",
			domainUserID: validID + "_" + validID + "_" + validID + "_" + validID,
			respDomainID: "",
			respUserID:   "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ar, er := auth.DecodeDomainUserID(tc.domainUserID)
			assert.Equal(t, tc.respUserID, er, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respUserID, er))
			assert.Equal(t, tc.respDomainID, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respDomainID, ar))
		})
	}
}

func newToken(t *testing.T, issuerName string, key auth.Key) jwt.Token {
	builder := jwt.NewBuilder()
	builder.
		Issuer(issuerName).
		IssuedAt(key.IssuedAt).
		Claim(tokenType, key.Type).
		Expiration(key.ExpiresAt)
	builder.Claim(roleField, key.Role)
	builder.Claim(VerifiedField, key.Verified)
	if key.Subject != "" {
		builder.Subject(key.Subject)
	}
	if key.ID != "" {
		builder.JwtID(key.ID)
	}
	tkn, err := builder.Build()
	assert.Nil(t, err, fmt.Sprintf("Building token expected to succeed: %s", err))
	return tkn
}

func signToken(t *testing.T, issuerName string, key auth.Key, parseToken bool) (string, jwt.Token, error) {
	tkn := newToken(t, issuerName, key)
	pKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return "", nil, err
	}
	pubKey := &pKey.PublicKey
	sTkn, err := jwt.Sign(tkn, jwt.WithKey(jwa.RS256, pKey))
	if err != nil {
		return "", nil, err
	}
	if !parseToken {
		return string(sTkn), nil, nil
	}
	pTkn, err := jwt.Parse(
		sTkn,
		jwt.WithValidate(true),
		jwt.WithKey(jwa.RS256, pubKey),
	)
	if err != nil {
		return "", nil, err
	}
	return string(sTkn), pTkn, nil
}
