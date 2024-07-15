// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secret      = "secret"
	email       = "test@example.com"
	id          = "testID"
	groupName   = "mgx"
	description = "Description"

	memberRelation  = "member"
	authoritiesObj  = "authorities"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	errIssueUser          = errors.New("failed to issue new login key")
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	errRetrieve           = errors.New("failed to retrieve key data")
	ErrExpiry             = errors.New("token is expired")
	errRollbackPolicy     = errors.New("failed to rollback policy")
	errAddPolicies        = errors.New("failed to add policies")
	errPlatform           = errors.New("invalid platform id")
	inValidToken          = "invalid"
	inValid               = "invalid"
	valid                 = "valid"
	domain                = auth.Domain{
		ID:         validID,
		Name:       groupName,
		Tags:       []string{"tag1", "tag2"},
		Alias:      "test",
		Permission: auth.AdminPermission,
		CreatedBy:  validID,
		UpdatedBy:  validID,
	}
)

var (
	krepo    *mocks.KeyRepository
	prepo    *mocks.PolicyAgent
	drepo    *mocks.DomainsRepository
	patsrepo *mocks.PATSRepository
	hasher   *mocks.Hasher
)

func newService() (auth.Service, string) {
	krepo = new(mocks.KeyRepository)
	prepo = new(mocks.PolicyAgent)
	drepo = new(mocks.DomainsRepository)
	patsrepo = new(mocks.PATSRepository)
	hasher = new(mocks.Hasher)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))
	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   id,
		Type:      auth.AccessKey,
		User:      email,
		Domain:    groupName,
	}
	token, _ := t.Issue(key)

	return auth.New(krepo, drepo, patsrepo, hasher, idProvider, t, prepo, loginDuration, refreshDuration, invalidDuration), token
}

func TestIssue(t *testing.T) {
	svc, accessToken := newService()

	n := jwt.New([]byte(secret))

	apikey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   id,
		Type:      auth.APIKey,
		User:      email,
		Domain:    groupName,
	}
	apiToken, err := n.Issue(apikey)
	assert.Nil(t, err, fmt.Sprintf("Issuing API key expected to succeed: %s", err))

	refreshkey := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   id,
		Type:      auth.RefreshKey,
		User:      email,
		Domain:    groupName,
	}
	refreshToken, err := n.Issue(refreshkey)
	assert.Nil(t, err, fmt.Sprintf("Issuing refresh key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   auth.Key
		token string
		err   error
	}{
		{
			desc: "issue recovery key",
			key: auth.Key{
				Type:     auth.RecoveryKey,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
	}

	for _, tc := range cases {
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}

	cases2 := []struct {
		desc                   string
		key                    auth.Key
		saveResponse           auth.Key
		retrieveByIDResponse   auth.Domain
		token                  string
		saveErr                error
		checkPolicyRequest     auth.PolicyReq
		checkPlatformPolicyReq auth.PolicyReq
		checkDomainPolicyReq   auth.PolicyReq
		checkPolicyErr         error
		checkPolicyErr1        error
		retreiveByIDErr        error
		err                    error
	}{
		{
			desc: "issue login key",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
			},
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "issue login key with domain",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "issue login key with failed check on platform admin",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			token: accessToken,
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPlatformPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
				Object:      groupName,
			},
			checkPolicyErr:       repoerr.ErrNotFound,
			retrieveByIDResponse: auth.Domain{},
			retreiveByIDErr:      repoerr.ErrNotFound,
			err:                  repoerr.ErrNotFound,
		},
		{
			desc: "issue login key with failed check on platform admin with enabled status",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			token: accessToken,
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPlatformPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr:       svcerr.ErrAuthorization,
			checkPolicyErr1:      svcerr.ErrAuthorization,
			retrieveByIDResponse: auth.Domain{Status: auth.EnabledStatus},
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc: "issue login key with membership permission",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			token: accessToken,
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPlatformPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr:       svcerr.ErrAuthorization,
			checkPolicyErr1:      svcerr.ErrAuthorization,
			retrieveByIDResponse: auth.Domain{Status: auth.EnabledStatus},
			err:                  svcerr.ErrAuthorization,
		},
		{
			desc: "issue login key with membership permission with failed  to authorize",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			token: accessToken,
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPlatformPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr:       svcerr.ErrAuthorization,
			checkPolicyErr1:      svcerr.ErrAuthorization,
			retrieveByIDResponse: auth.Domain{Status: auth.EnabledStatus},
			err:                  svcerr.ErrAuthorization,
		},
	}
	for _, tc := range cases2 {
		repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyRequest).Return(tc.checkPolicyErr)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, tc.checkPlatformPolicyReq).Return(tc.checkPolicyErr1)
		repoCall3 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(tc.retrieveByIDResponse, tc.retreiveByIDErr)
		repoCall4 := prepo.On("CheckPolicy", mock.Anything, tc.checkDomainPolicyReq).Return(tc.checkPolicyErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}

	cases3 := []struct {
		desc    string
		key     auth.Key
		token   string
		saveErr error
		err     error
	}{
		{
			desc: "issue API key",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "issue API key with an invalid token",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: "invalid",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: " issue API key with invalid key request",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: apiToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "issue API key with failed to save",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token:   accessToken,
			saveErr: repoerr.ErrNotFound,
			err:     repoerr.ErrNotFound,
		},
	}
	for _, tc := range cases3 {
		repoCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}

	cases4 := []struct {
		desc                 string
		key                  auth.Key
		token                string
		checkPolicyRequest   auth.PolicyReq
		checkDOmainPolicyReq auth.PolicyReq
		checkPolicyErr       error
		retrieveByIDErr      error
		err                  error
	}{
		{
			desc: "issue refresh key",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
			},
			checkPolicyRequest: auth.PolicyReq{
				Subject:     email,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			token: refreshToken,
			err:   nil,
		},
		{
			desc: "issue refresh token with invalid policy",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
			},
			checkPolicyRequest: auth.PolicyReq{
				Subject:     email,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDOmainPolicyReq: auth.PolicyReq{
				Subject:     "mgx_test@example.com",
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token:           refreshToken,
			checkPolicyErr:  svcerr.ErrAuthorization,
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc: "issue refresh key with invalid token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
			},
			checkDOmainPolicyReq: auth.PolicyReq{
				Subject:     "mgx_test@example.com",
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token: accessToken,
			err:   errIssueUser,
		},
		{
			desc: "issue refresh key with empty token",
			key: auth.Key{
				Type:     auth.RefreshKey,
				IssuedAt: time.Now(),
			},
			checkDOmainPolicyReq: auth.PolicyReq{
				Subject:     "mgx_test@example.com",
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token: "",
			err:   errRetrieve,
		},
		{
			desc: "issue invitation key",
			key: auth.Key{
				Type:     auth.InvitationKey,
				IssuedAt: time.Now(),
			},
			checkPolicyRequest: auth.PolicyReq{
				Subject:     email,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			token: "",
			err:   nil,
		},
		{
			desc: "issue invitation key with invalid policy",
			key: auth.Key{
				Type:     auth.InvitationKey,
				IssuedAt: time.Now(),
				Domain:   groupName,
			},
			checkPolicyRequest: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDOmainPolicyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			token:           refreshToken,
			checkPolicyErr:  svcerr.ErrAuthorization,
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrDomainAuthorization,
		},
	}
	for _, tc := range cases4 {
		repoCall := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyRequest).Return(tc.checkPolicyErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.retrieveByIDErr)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, tc.checkDOmainPolicyReq).Return(tc.checkPolicyErr)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestRevoke(t *testing.T) {
	svc, _ := newService()
	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, errIssueUser)
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	repocall.Unset()
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall1 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	key := auth.Key{
		Type:     auth.APIKey,
		IssuedAt: time.Now(),
		Subject:  id,
	}
	_, err = svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))
	repocall1.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "revoke login key",
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc:  "revoke non-existing login key",
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc:  "revoke with empty login key",
			token: "",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "revoke login key with failed to remove",
			id:    "invalidID",
			token: secret.AccessToken,
			err:   svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestRetrieve(t *testing.T) {
	svc, _ := newService()
	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()
	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  id,
		IssuedAt: time.Now(),
	}

	repocall1 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall1.Unset()

	repocall2 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login's key expected to succeed: %s", err))
	repocall2.Unset()

	repocall3 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	resetToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))
	repocall3.Unset()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "retrieve login key",
			token: userToken.AccessToken,
			err:   nil,
		},
		{
			desc:  "retrieve non-existing login key",
			id:    "invalid",
			token: userToken.AccessToken,
			err:   svcerr.ErrNotFound,
		},
		{
			desc:  "retrieve with wrong login key",
			token: "wrong",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "retrieve with API token",
			token: apiToken.AccessToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc:  "retrieve with reset token",
			token: resetToken.AccessToken,
			err:   svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
		_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestIdentify(t *testing.T) {
	svc, _ := newService()

	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repocall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, User: id, IssuedAt: time.Now(), Domain: groupName})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()
	repocall1.Unset()

	repocall2 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))
	repocall2.Unset()

	repocall3 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	apiSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, Subject: id, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall3.Unset()

	repocall4 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	exp0 := time.Now().UTC().Add(-10 * time.Second).Round(time.Second)
	exp1 := time.Now().UTC().Add(-1 * time.Minute).Round(time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, IssuedAt: exp0, ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))
	repocall4.Unset()

	te := jwt.New([]byte(secret))
	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   id,
		Type:      7,
		User:      email,
		Domain:    groupName,
	}
	invalidTokenType, _ := te.Issue(key)

	cases := []struct {
		desc string
		key  string
		idt  string
		err  error
	}{
		{
			desc: "identify login key",
			key:  loginSecret.AccessToken,
			idt:  id,
			err:  nil,
		},
		{
			desc: "identify refresh key",
			key:  loginSecret.RefreshToken,
			idt:  id,
			err:  nil,
		},
		{
			desc: "identify recovery key",
			key:  recoverySecret.AccessToken,
			idt:  id,
			err:  nil,
		},
		{
			desc: "identify API key",
			key:  apiSecret.AccessToken,
			idt:  id,
			err:  nil,
		},
		{
			desc: "identify expired API key",
			key:  expSecret.AccessToken,
			idt:  "",
			err:  auth.ErrKeyExpired,
		},
		{
			desc: "identify API key with failed to retrieve",
			key:  apiSecret.AccessToken,
			idt:  "",
			err:  svcerr.ErrAuthentication,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  "",
			err:  svcerr.ErrAuthentication,
		},
		{
			desc: "identify invalid key type",
			key:  invalidTokenType,
			idt:  "",
			err:  svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
		repocall1 := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(tc.err)
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt.Subject, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
		repocall.Unset()
		repocall1.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	svc, accessToken := newService()

	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repocall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, User: id, IssuedAt: time.Now(), Domain: groupName})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()
	repocall1.Unset()
	saveCall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	exp1 := time.Now().Add(-2 * time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))
	saveCall.Unset()

	repocall2 := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	repocall3 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	emptySubject, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, User: "", IssuedAt: time.Now(), Domain: groupName})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall2.Unset()
	repocall3.Unset()

	te := jwt.New([]byte(secret))
	key := auth.Key{
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(refreshDuration),
		Subject:   id,
		Type:      auth.AccessKey,
		User:      email,
	}
	emptyDomain, _ := te.Issue(key)

	cases := []struct {
		desc                 string
		policyReq            auth.PolicyReq
		retrieveDomainRes    auth.Domain
		checkPolicyReq3      auth.PolicyReq
		checkAdminPolicyReq  auth.PolicyReq
		checkDomainPolicyReq auth.PolicyReq
		checkPolicyErr       error
		checkPolicyErr1      error
		checkPolicyErr2      error
		err                  error
	}{
		{
			desc: "authorize token successfully",
			policyReq: auth.PolicyReq{
				Subject:     accessToken,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: nil,
		},
		{
			desc: "authorize token for group type with empty domain",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      "",
				ObjectType:  auth.GroupType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      "",
				ObjectType:  auth.GroupType,
				Permission:  auth.AdminPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err:            svcerr.ErrDomainAuthorization,
			checkPolicyErr: svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize token with disabled domain",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Permission:  auth.AdminPermission,
				Object:      validID,
				ObjectType:  auth.DomainType,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},

			retrieveDomainRes: auth.Domain{
				ID:     validID,
				Name:   groupName,
				Status: auth.DisabledStatus,
			},
			err: nil,
		},
		{
			desc: "authorize token with disabled domain with failed to authorize",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Permission:  auth.AdminPermission,
				Object:      validID,
				ObjectType:  auth.DomainType,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},

			retrieveDomainRes: auth.Domain{
				ID:     validID,
				Name:   groupName,
				Status: auth.DisabledStatus,
			},
			checkPolicyErr1: svcerr.ErrDomainAuthorization,
			err:             svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize token with frozen domain",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Permission:  auth.AdminPermission,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},

			retrieveDomainRes: auth.Domain{
				ID:     validID,
				Name:   groupName,
				Status: auth.FreezeStatus,
			},
			err: nil,
		},
		{
			desc: "authorize token with frozen domain with failed to authorize",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Permission:  auth.AdminPermission,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},

			retrieveDomainRes: auth.Domain{
				ID:     validID,
				Name:   groupName,
				Status: auth.FreezeStatus,
			},
			checkPolicyErr1: svcerr.ErrDomainAuthorization,
			err:             svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize token with domain with invalid status",
			policyReq: auth.PolicyReq{
				Subject:     emptyDomain,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Permission:  auth.AdminPermission,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},

			retrieveDomainRes: auth.Domain{
				ID:     validID,
				Name:   groupName,
				Status: auth.AllStatus,
			},
			err: svcerr.ErrDomainAuthorization,
		},

		{
			desc: "authorize an expired token",
			policyReq: auth.PolicyReq{
				Subject:     expSecret.AccessToken,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc: "authorize a token with an empty subject",
			policyReq: auth.PolicyReq{
				Subject:     emptySubject.AccessToken,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc: "authorize a token with an empty secret and invalid type",
			policyReq: auth.PolicyReq{
				Subject:     emptySubject.AccessToken,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformKind,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: svcerr.ErrDomainAuthorization,
		},
		{
			desc: "authorize a user key successfully",
			policyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: nil,
		},
		{
			desc: "authorize token with empty subject and domain object type",
			policyReq: auth.PolicyReq{
				Subject:     emptySubject.AccessToken,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkPolicyReq3: auth.PolicyReq{
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			err: svcerr.ErrDomainAuthorization,
		},
	}
	for _, tc := range cases {
		repoCall := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyReq3).Return(tc.checkPolicyErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(tc.retrieveDomainRes, nil)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, tc.checkAdminPolicyReq).Return(tc.checkPolicyErr1)
		repoCall3 := prepo.On("CheckPolicy", mock.Anything, tc.checkDomainPolicyReq).Return(tc.checkPolicyErr1)
		repoCall4 := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		err := svc.Authorize(context.Background(), tc.policyReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
	cases2 := []struct {
		desc      string
		policyReq auth.PolicyReq
		err       error
	}{
		{
			desc: "authorize token with invalid platform validation",
			policyReq: auth.PolicyReq{
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      validID,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			err: errPlatform,
		},
	}
	for _, tc := range cases2 {
		err := svc.Authorize(context.Background(), tc.policyReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAddPolicy(t *testing.T) {
	svc, _ := newService()

	cases := []struct {
		desc string
		pr   auth.PolicyReq
		err  error
	}{
		{
			desc: "add policy successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			err: nil,
		},
		{
			desc: "add policy with invalid object",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      inValid,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			err: svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repocall := prepo.On("AddPolicy", mock.Anything, mock.Anything).Return(tc.err)
		err := svc.AddPolicy(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestAddPolicies(t *testing.T) {
	svc, _ := newService()

	cases := []struct {
		desc string
		pr   []auth.PolicyReq
		err  error
	}{
		{
			desc: "add policy successfully",
			pr: []auth.PolicyReq{
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
			},
			err: nil,
		},
		{
			desc: "add policy with invalid object",
			pr: []auth.PolicyReq{
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      inValid,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
			},
			err: svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.err)
		err := svc.AddPolicies(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestDeletePolicy(t *testing.T) {
	svc, _ := newService()

	cases := []struct {
		desc string
		pr   auth.PolicyReq
		err  error
	}{
		{
			desc: "delete policy successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			err: nil,
		},
		{
			desc: "delete policy with invalid object",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.UsersKind,
				Object:      inValid,
				ObjectType:  auth.PlatformType,
				Permission:  auth.AdminPermission,
			},
			err: svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repocall := prepo.On("DeletePolicyFilter", context.Background(), mock.Anything).Return(tc.err)
		err := svc.DeletePolicyFilter(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestDeletePolicies(t *testing.T) {
	svc, _ := newService()

	cases := []struct {
		desc string
		pr   []auth.PolicyReq
		err  error
	}{
		{
			desc: "delete policy successfully",
			pr: []auth.PolicyReq{
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
			},
			err: nil,
		},
		{
			desc: "delete policy with invalid object",
			pr: []auth.PolicyReq{
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      inValid,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
				{
					Subject:     id,
					SubjectType: auth.UserType,
					SubjectKind: auth.UsersKind,
					Object:      auth.MagistralaObject,
					ObjectType:  auth.PlatformType,
					Permission:  auth.AdminPermission,
				},
			},
			err: svcerr.ErrInvalidPolicy,
		},
	}

	for _, tc := range cases {
		repocall := prepo.On("DeletePolicies", context.Background(), mock.Anything).Return(tc.err)
		err := svc.DeletePolicies(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestListObjects(t *testing.T) {
	svc, accessToken := newService()

	pageLen := 15
	expectedPolicies := make([]auth.PolicyRes, pageLen)

	cases := []struct {
		desc          string
		pr            auth.PolicyReq
		nextPageToken string
		limit         uint64
		err           error
	}{
		{
			desc: "list objects successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      "",
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           nil,
		},
		{
			desc: "list objects with invalid request",
			pr: auth.PolicyReq{
				Subject:     inValid,
				SubjectType: inValid,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      inValid,
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           svcerr.ErrInvalidPolicy,
		},
	}
	for _, tc := range cases {
		repocall2 := prepo.On("RetrieveObjects", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, tc.err)
		page, err := svc.ListObjects(context.Background(), tc.pr, tc.nextPageToken, tc.limit)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("listing policies expected to succeed: %s", err))
		if err == nil {
			assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
		}
		repocall2.Unset()
	}
}

func TestListAllObjects(t *testing.T) {
	svc, accessToken := newService()

	pageLen := 15
	expectedPolicies := make([]auth.PolicyRes, pageLen)

	cases := []struct {
		desc          string
		pr            auth.PolicyReq
		nextPageToken string
		limit         int32
		err           error
	}{
		{
			desc: "list all objects successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      "",
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           nil,
		},
		{
			desc: "list all objects with invalid request",
			pr: auth.PolicyReq{
				Subject:     inValid,
				SubjectType: inValid,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      inValid,
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           svcerr.ErrInvalidPolicy,
		},
	}
	for _, tc := range cases {
		repocall2 := prepo.On("RetrieveAllObjects", context.Background(), mock.Anything).Return(expectedPolicies, tc.err)
		page, err := svc.ListAllObjects(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("listing policies expected to succeed: %s", err))
		if err == nil {
			assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
		}
		repocall2.Unset()
	}
}

func TestCountObjects(t *testing.T) {
	svc, _ := newService()

	pageLen := uint64(15)

	repocall2 := prepo.On("RetrieveAllObjectsCount", context.Background(), mock.Anything, mock.Anything).Return(pageLen, nil)
	count, err := svc.CountObjects(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("counting policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, count, fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, count, err))
	repocall2.Unset()
}

func TestListSubjects(t *testing.T) {
	svc, accessToken := newService()

	pageLen := 15
	expectedPolicies := make([]auth.PolicyRes, pageLen)

	cases := []struct {
		desc          string
		pr            auth.PolicyReq
		nextPageToken string
		limit         uint64
		err           error
	}{
		{
			desc: "list subjects successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      "",
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           nil,
		},
		{
			desc: "list subjects with invalid request",
			pr: auth.PolicyReq{
				Subject:     inValid,
				SubjectType: inValid,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      inValid,
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           svcerr.ErrInvalidPolicy,
		},
	}
	for _, tc := range cases {
		repocall := prepo.On("RetrieveSubjects", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, tc.err)
		page, err := svc.ListSubjects(context.Background(), tc.pr, tc.nextPageToken, tc.limit)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("listing policies expected to succeed: %s", err))
		if err == nil {
			assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
		}
		repocall.Unset()
	}
}

func TestListAllSubjects(t *testing.T) {
	svc, accessToken := newService()

	pageLen := 15
	expectedPolicies := make([]auth.PolicyRes, pageLen)

	cases := []struct {
		desc          string
		pr            auth.PolicyReq
		nextPageToken string
		limit         int32
		err           error
	}{
		{
			desc: "list all subjects successfully",
			pr: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      "",
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           nil,
		},
		{
			desc: "list all subjects with invalid request",
			pr: auth.PolicyReq{
				Subject:     inValid,
				SubjectType: inValid,
				Relation:    auth.ContributorRelation,
				ObjectType:  auth.ThingType,
				ObjectKind:  auth.ThingsKind,
				Object:      inValid,
			},
			nextPageToken: accessToken,
			limit:         10,
			err:           svcerr.ErrInvalidPolicy,
		},
	}
	for _, tc := range cases {
		repocall := prepo.On("RetrieveAllSubjects", context.Background(), mock.Anything).Return(expectedPolicies, tc.err)
		page, err := svc.ListAllSubjects(context.Background(), tc.pr)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("listing policies expected to succeed: %s", err))
		if err == nil {
			assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
		}
		repocall.Unset()
	}
}

func TestCountSubjects(t *testing.T) {
	svc, _ := newService()
	pageLen := uint64(15)

	repocall := prepo.On("RetrieveAllSubjectsCount", mock.Anything, mock.Anything, mock.Anything).Return(pageLen, nil)
	count, err := svc.CountSubjects(context.Background(), auth.PolicyReq{Object: id, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("counting policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, count, fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, count, err))
	repocall.Unset()
}

func TestListPermissions(t *testing.T) {
	svc, _ := newService()

	pr := auth.PolicyReq{
		Subject:     id,
		SubjectType: auth.UserType,
		Relation:    auth.ContributorRelation,
		ObjectType:  auth.ThingType,
		ObjectKind:  auth.ThingsKind,
		Object:      "",
	}
	filterPermisions := []string{auth.ViewPermission, auth.AdminPermission}

	repoCall := prepo.On("RetrievePermissions", context.Background(), pr, filterPermisions).Return(auth.Permissions{}, nil)
	_, err := svc.ListPermissions(context.Background(), pr, filterPermisions)
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	repoCall.Unset()
}

func TestSwitchToPermission(t *testing.T) {
	cases := []struct {
		desc     string
		relation string
		result   string
	}{
		{
			desc:     "switch to admin permission",
			relation: auth.AdministratorRelation,
			result:   auth.AdminPermission,
		},
		{
			desc:     "switch to editor permission",
			relation: auth.EditorRelation,
			result:   auth.EditPermission,
		},
		{
			desc:     "switch to contributor permission",
			relation: auth.ContributorRelation,
			result:   auth.ViewPermission,
		},
		{
			desc:     "switch to member permission",
			relation: auth.MemberRelation,
			result:   auth.MembershipPermission,
		},
		{
			desc:     "switch to group permission",
			relation: auth.GroupRelation,
			result:   auth.GroupRelation,
		},
		{
			desc:     "switch to guest permission",
			relation: auth.GuestRelation,
			result:   auth.ViewPermission,
		},
	}
	for _, tc := range cases {
		result := auth.SwitchToPermission(tc.relation)
		assert.Equal(t, tc.result, result, fmt.Sprintf("switching to permission expected to succeed: %s", result))
	}
}

func TestCreateDomain(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc              string
		d                 auth.Domain
		token             string
		userID            string
		addPolicyErr      error
		savePolicyErr     error
		saveDomainErr     error
		deleteDomainErr   error
		deletePoliciesErr error
		err               error
	}{
		{
			desc: "create domain successfully",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token: accessToken,
			err:   nil,
		},
		{
			desc: "create domain with invalid token",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token: inValidToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "create domain with invalid status",
			d: auth.Domain{
				Status: auth.AllStatus,
			},
			token: accessToken,
			err:   svcerr.ErrInvalidStatus,
		},
		{
			desc: "create domain with failed policy request",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:        accessToken,
			addPolicyErr: errors.ErrMalformedEntity,
			err:          errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed save policy request",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:         accessToken,
			savePolicyErr: errors.ErrMalformedEntity,
			err:           errCreateDomainPolicy,
		},
		{
			desc: "create domain with failed save domain request",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:         accessToken,
			saveDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrCreateEntity,
		},
		{
			desc: "create domain with rollback error",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:           accessToken,
			savePolicyErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with rollback error and failed to delete policies",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:             accessToken,
			savePolicyErr:     errors.ErrMalformedEntity,
			deleteDomainErr:   errors.ErrMalformedEntity,
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errors.ErrMalformedEntity,
		},
		{
			desc: "create domain with failed to create and failed rollback",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:             accessToken,
			saveDomainErr:     errors.ErrMalformedEntity,
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errRollbackPolicy,
		},
		{
			desc: "create domain with failed to create and failed rollback",
			d: auth.Domain{
				Status: auth.EnabledStatus,
			},
			token:           accessToken,
			saveDomainErr:   errors.ErrMalformedEntity,
			deleteDomainErr: errors.ErrMalformedEntity,
			err:             errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		repoCall1 := drepo.On("SavePolicies", mock.Anything, mock.Anything).Return(tc.savePolicyErr)
		repoCall2 := prepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		repoCall3 := drepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deleteDomainErr)
		repoCall4 := drepo.On("Save", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.saveDomainErr)
		_, err := svc.CreateDomain(context.Background(), tc.token, tc.d)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc           string
		token          string
		domainID       string
		domainRepoErr  error
		domainRepoErr1 error
		checkPolicyErr error
		err            error
	}{
		{
			desc:     "retrieve domain successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "retrieve domain with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:           "retrieve domain with empty domain id",
			token:          accessToken,
			domainID:       "",
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve non-existing domain",
			token:          accessToken,
			domainID:       inValid,
			domainRepoErr:  repoerr.ErrNotFound,
			err:            svcerr.ErrViewEntity,
			domainRepoErr1: repoerr.ErrNotFound,
		},
		{
			desc:           "retrieve domain with failed to retrieve by id",
			token:          accessToken,
			domainID:       validID,
			domainRepoErr1: repoerr.ErrNotFound,
			err:            svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, groupName).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		repoCall2 := drepo.On("RetrieveByID", mock.Anything, tc.domainID).Return(auth.Domain{}, tc.domainRepoErr1)
		_, err := svc.RetrieveDomain(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestRetrieveDomainPermissions(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc                   string
		token                  string
		domainID               string
		retreivePermissionsErr error
		retreiveByIDErr        error
		checkPolicyErr         error
		err                    error
	}{
		{
			desc:     "retrieve domain permissions successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "retrieve domain permissions with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:           "retrieve domain permissions with empty domainID",
			token:          accessToken,
			domainID:       "",
			checkPolicyErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrDomainAuthorization,
		},
		{
			desc:                   "retrieve domain permissions with failed to retrieve permissions",
			token:                  accessToken,
			domainID:               validID,
			retreivePermissionsErr: repoerr.ErrNotFound,
			err:                    svcerr.ErrNotFound,
		},
		{
			desc:            "retrieve domain permissions with failed to retrieve by id",
			token:           accessToken,
			domainID:        validID,
			retreiveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("RetrievePermissions", mock.Anything, mock.Anything, mock.Anything).Return(auth.Permissions{}, tc.retreivePermissionsErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.retreiveByIDErr)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		_, err := svc.RetrieveDomainPermissions(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateDomain(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc            string
		token           string
		domainID        string
		domReq          auth.DomainReq
		checkPolicyErr  error
		retrieveByIDErr error
		updateErr       error
		err             error
	}{
		{
			desc:     "update domain successfully",
			token:    accessToken,
			domainID: validID,
			domReq: auth.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: nil,
		},
		{
			desc:     "update domain with invalid token",
			token:    inValidToken,
			domainID: validID,
			domReq: auth.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "update domain with empty domainID",
			token:    accessToken,
			domainID: "",
			domReq: auth.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			checkPolicyErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "update domain with failed to retrieve by id",
			token:    accessToken,
			domainID: validID,
			domReq: auth.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			retrieveByIDErr: repoerr.ErrNotFound,
			err:             svcerr.ErrNotFound,
		},
		{
			desc:     "update domain with failed to update",
			token:    accessToken,
			domainID: validID,
			domReq: auth.DomainReq{
				Name:  &valid,
				Alias: &valid,
			},
			updateErr: errors.ErrMalformedEntity,
			err:       errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.retrieveByIDErr)
		repoCall2 := drepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.Domain{}, tc.updateErr)
		_, err := svc.UpdateDomain(context.Background(), tc.token, tc.domainID, tc.domReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestChangeDomainStatus(t *testing.T) {
	svc, accessToken := newService()

	disabledStatus := auth.DisabledStatus

	cases := []struct {
		desc             string
		token            string
		domainID         string
		domainReq        auth.DomainReq
		retreieveByIDErr error
		checkPolicyErr   error
		updateErr        error
		err              error
	}{
		{
			desc:     "change domain status successfully",
			token:    accessToken,
			domainID: validID,
			domainReq: auth.DomainReq{
				Status: &disabledStatus,
			},
			err: nil,
		},
		{
			desc:     "change domain status with invalid token",
			token:    inValidToken,
			domainID: validID,
			domainReq: auth.DomainReq{
				Status: &disabledStatus,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "change domain status with empty domainID",
			token:    accessToken,
			domainID: "",
			domainReq: auth.DomainReq{
				Status: &disabledStatus,
			},
			retreieveByIDErr: repoerr.ErrNotFound,
			err:              svcerr.ErrNotFound,
		},
		{
			desc:     "change domain status with unauthorized domain ID",
			token:    accessToken,
			domainID: validID,
			domainReq: auth.DomainReq{
				Status: &disabledStatus,
			},
			checkPolicyErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "change domain status with repository error on update",
			token:    accessToken,
			domainID: validID,
			domainReq: auth.DomainReq{
				Status: &disabledStatus,
			},
			updateErr: errors.ErrMalformedEntity,
			err:       errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.retreieveByIDErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		repoCall2 := drepo.On("Update", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(auth.Domain{}, tc.updateErr)
		_, err := svc.ChangeDomainStatus(context.Background(), tc.token, tc.domainID, tc.domainReq)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestListDomains(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc            string
		token           string
		domainID        string
		authReq         auth.Page
		listDomainsRes  auth.DomainsPage
		retreiveByIDErr error
		checkPolicyErr  error
		listDomainErr   error
		err             error
	}{
		{
			desc:     "list domains successfully",
			token:    accessToken,
			domainID: validID,
			authReq: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     auth.EnabledStatus,
			},
			listDomainsRes: auth.DomainsPage{
				Domains: []auth.Domain{domain},
			},
			err: nil,
		},
		{
			desc:     "list domains with invalid token",
			token:    inValidToken,
			domainID: validID,
			authReq: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     auth.EnabledStatus,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "list domains with repository error on list domains",
			token:    accessToken,
			domainID: validID,
			authReq: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
				Status:     auth.EnabledStatus,
			},
			listDomainErr: errors.ErrMalformedEntity,
			err:           svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		repoCall1 := drepo.On("ListDomains", mock.Anything, mock.Anything).Return(tc.listDomainsRes, tc.listDomainErr)
		_, err := svc.ListDomains(context.Background(), tc.token, auth.Page{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestAssignUsers(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc                 string
		token                string
		domainID             string
		userIDs              []string
		relation             string
		checkPolicyReq3      auth.PolicyReq
		checkAdminPolicyReq  auth.PolicyReq
		checkDomainPolicyReq auth.PolicyReq
		checkPolicyReq33     auth.PolicyReq
		checkpolicyErr       error
		checkPolicyErr1      error
		checkPolicyErr2      error
		addPoliciesErr       error
		savePoliciesErr      error
		deletePoliciesErr    error
		err                  error
	}{
		{
			desc:     "assign users successfully",
			token:    accessToken,
			domainID: validID,
			userIDs:  []string{validID},
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     validID,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},

			err: nil,
		},
		{
			desc:     "assign users with invalid token",
			token:    inValidToken,
			domainID: validID,
			userIDs:  []string{validID},
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     validID,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "assign users with invalid domainID",
			token:    accessToken,
			domainID: inValid,
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      inValid,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      inValid,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr1: svcerr.ErrAuthorization,
			err:             svcerr.ErrAuthorization,
		},
		{
			desc:     "assign users with invalid userIDs",
			token:    accessToken,
			userIDs:  []string{inValid},
			domainID: validID,
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     inValid,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr2: svcerr.ErrMalformedEntity,
			err:             svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "assign users with failed to add policies to agent",
			token:    accessToken,
			domainID: validID,
			userIDs:  []string{validID},
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     validID,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			addPoliciesErr: svcerr.ErrAuthorization,
			err:            errAddPolicies,
		},
		{
			desc:     "assign users with failed to save policies to domain",
			token:    accessToken,
			domainID: validID,
			userIDs:  []string{validID},
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     validID,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			savePoliciesErr: repoerr.ErrCreateEntity,
			err:             errAddPolicies,
		},
		{
			desc:     "assign users with failed to save policies to domain and failed to delete",
			token:    accessToken,
			domainID: validID,
			userIDs:  []string{validID},
			relation: auth.ContributorRelation,
			checkPolicyReq3: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.ViewPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     validID,
				SubjectType: auth.UserType,
				Object:      auth.MagistralaObject,
				ObjectType:  auth.PlatformType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyReq33: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			savePoliciesErr:   repoerr.ErrCreateEntity,
			deletePoliciesErr: svcerr.ErrDomainAuthorization,
			err:               errAddPolicies,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, groupName).Return(auth.Domain{}, nil)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyReq3).Return(tc.checkpolicyErr)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, tc.checkAdminPolicyReq).Return(tc.checkPolicyErr1)
		repoCall3 := prepo.On("CheckPolicy", mock.Anything, tc.checkDomainPolicyReq).Return(tc.checkPolicyErr2)
		repoCall4 := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyReq33).Return(tc.checkPolicyErr2)
		repoCall5 := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPoliciesErr)
		repoCall6 := drepo.On("SavePolicies", mock.Anything, mock.Anything, mock.Anything).Return(tc.savePoliciesErr)
		repoCall7 := prepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		err := svc.AssignUsers(context.Background(), tc.token, tc.domainID, tc.userIDs, tc.relation)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		repoCall5.Unset()
		repoCall6.Unset()
		repoCall7.Unset()
	}
}

func TestUnassignUser(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc                  string
		token                 string
		domainID              string
		userID                string
		checkPolicyReq        auth.PolicyReq
		checkAdminPolicyReq   auth.PolicyReq
		checkDomainPolicyReq  auth.PolicyReq
		checkPolicyErr        error
		checkPolicyErr1       error
		deletePolicyFilterErr error
		deletePoliciesErr     error
		err                   error
	}{
		{
			desc:     "unassign user successfully",
			token:    accessToken,
			domainID: validID,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			err: nil,
		},
		{
			desc:     "unassign users with invalid token",
			token:    inValidToken,
			domainID: validID,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:     "unassign users with invalid domainID",
			token:    accessToken,
			domainID: inValid,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      inValid,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      inValid,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkPolicyErr1: svcerr.ErrAuthorization,
			err:             svcerr.ErrDomainAuthorization,
		},
		{
			desc:     "unassign users with failed to delete policies from agent",
			token:    accessToken,
			domainID: validID,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			deletePolicyFilterErr: errors.ErrMalformedEntity,
			err:                   errors.ErrMalformedEntity,
		},
		{
			desc:     "unassign users with failed to delete policies from domain",
			token:    accessToken,
			domainID: validID,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			deletePoliciesErr:     errors.ErrMalformedEntity,
			deletePolicyFilterErr: errors.ErrMalformedEntity,
			err:                   errors.ErrMalformedEntity,
		},
		{
			desc:     "unassign user with failed to delete policy from domain",
			token:    accessToken,
			domainID: validID,
			userID:   validID,
			checkPolicyReq: auth.PolicyReq{
				Subject:     id,
				SubjectType: auth.UserType,
				Object:      groupName,
				ObjectType:  auth.DomainType,
				Permission:  auth.MembershipPermission,
			},
			checkAdminPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.AdminPermission,
			},
			checkDomainPolicyReq: auth.PolicyReq{
				Domain:      groupName,
				Subject:     id,
				SubjectType: auth.UserType,
				SubjectKind: auth.TokenKind,
				Object:      validID,
				ObjectType:  auth.DomainType,
				Permission:  auth.SharePermission,
			},
			deletePoliciesErr: errors.ErrMalformedEntity,
			err:               errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, nil)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, tc.checkPolicyReq).Return(tc.checkPolicyErr)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, tc.checkAdminPolicyReq).Return(tc.checkPolicyErr1)
		repoCall3 := prepo.On("CheckPolicy", mock.Anything, tc.checkDomainPolicyReq).Return(tc.checkPolicyErr1)
		repoCall4 := prepo.On("DeletePolicyFilter", mock.Anything, mock.Anything).Return(tc.deletePolicyFilterErr)
		repoCall5 := drepo.On("DeletePolicies", mock.Anything, mock.Anything, mock.Anything).Return(tc.deletePoliciesErr)
		err := svc.UnassignUser(context.Background(), tc.token, tc.domainID, tc.userID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
		repoCall5.Unset()
	}
}

func TestListUsersDomains(t *testing.T) {
	svc, accessToken := newService()

	cases := []struct {
		desc            string
		token           string
		userID          string
		page            auth.Page
		retreiveByIDErr error
		checkPolicyErr  error
		listDomainErr   error
		err             error
	}{
		{
			desc:   "list users domains successfully",
			token:  accessToken,
			userID: validID,
			page: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
			},
			err: nil,
		},
		{
			desc:   "list users domains successfully was admin",
			token:  accessToken,
			userID: email,
			page: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
			},
			err: nil,
		},
		{
			desc:   "list users domains with invalid token",
			token:  inValidToken,
			userID: validID,
			page: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
			},
			err: svcerr.ErrAuthentication,
		},
		{
			desc:   "list users domains with invalid domainID",
			token:  accessToken,
			userID: inValid,
			page: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
			},
			checkPolicyErr: svcerr.ErrAuthorization,
			err:            svcerr.ErrAuthorization,
		},
		{
			desc:   "list users domains with repository error on list domains",
			token:  accessToken,
			userID: validID,
			page: auth.Page{
				Offset:     0,
				Limit:      10,
				Permission: auth.AdminPermission,
			},
			listDomainErr: repoerr.ErrNotFound,
			err:           svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(tc.checkPolicyErr)
		repoCall1 := drepo.On("ListDomains", mock.Anything, mock.Anything).Return(auth.DomainsPage{}, tc.listDomainErr)
		_, err := svc.ListUserDomains(context.Background(), tc.token, tc.userID, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
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
		ar := auth.EncodeDomainUserID(tc.domainID, tc.userID)
		assert.Equal(t, tc.response, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.response, ar))
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
		ar, er := auth.DecodeDomainUserID(tc.domainUserID)
		assert.Equal(t, tc.respUserID, er, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respUserID, er))
		assert.Equal(t, tc.respDomainID, ar, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.respDomainID, ar))
	}
}
