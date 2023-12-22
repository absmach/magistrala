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
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
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
)

var (
	errIssueUser = errors.New("failed to issue new login key")
	// ErrExpiry indicates that the token is expired.
	ErrExpiry = errors.New("token is expired")
)

func newService() (auth.Service, *mocks.KeyRepository, string, *mocks.PolicyAgent) {
	krepo := new(mocks.KeyRepository)
	prepo := new(mocks.PolicyAgent)
	drepo := new(mocks.DomainsRepository)
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

	return auth.New(krepo, drepo, idProvider, t, prepo, loginDuration, refreshDuration, invalidDuration), krepo, token, prepo
}

func TestIssue(t *testing.T) {
	svc, krepo, accessToken, _ := newService()

	cases := []struct {
		desc  string
		key   auth.Key
		token string
		err   error
	}{
		{
			desc: "issue login key",
			key: auth.Key{
				Type:     auth.AccessKey,
				IssuedAt: time.Now(),
			},
			token: accessToken,
			err:   nil,
		},
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
		repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, tc.err)
		_, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repocall.Unset()
	}
}

func TestRevoke(t *testing.T) {
	svc, krepo, _, _ := newService()
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
			desc: "revoke login key",
			// id:    newKey.ID,
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc: "revoke non-existing login key",
			// id:    newKey.ID,
			token: secret.AccessToken,
			err:   nil,
		},
		{
			desc: "revoke with empty login key",
			// id:    newKey.ID,
			token: "",
			err:   svcerr.ErrAuthentication,
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
	svc, krepo, _, _ := newService()
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
			desc: "retrieve login key",
			// id:    apiKey.ID,
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
			desc: "retrieve with wrong login key",
			// id:    apiKey.ID,
			token: "wrong",
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "retrieve with API token",
			// id:    apiKey.ID,
			token: apiToken.AccessToken,
			err:   svcerr.ErrAuthentication,
		},
		{
			desc: "retrieve with reset token",
			// id:    apiKey.ID,
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
	svc, krepo, _, prepo := newService()

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
	exp1 := time.Now().Add(-2 * time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))
	repocall4.Unset()

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
			err:  ErrExpiry,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  "",
			err:  errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(auth.Key{}, tc.err)
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt.Subject, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
		repocall.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	svc, _, _, prepo := newService()

	repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	pr := auth.PolicyReq{Object: authoritiesObj, Relation: memberRelation, Subject: id}
	err := svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("authorizing initial %v policy expected to succeed: %s", pr, err))
	repocall.Unset()
}

func TestAddPolicy(t *testing.T) {
	svc, _, _, prepo := newService()

	repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(nil)
	prs := []auth.PolicyReq{{Object: "obj", ObjectType: "object", Relation: "rel", Subject: "sub", SubjectType: "subject"}}
	err := svc.AddPolicies(context.Background(), prs)
	require.Nil(t, err, fmt.Sprintf("adding %v policies expected to succeed: %v", prs, err))
	repocall.Unset()
	for _, pr := range prs {
		repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		err = svc.Authorize(context.Background(), pr)
		require.Nil(t, err, fmt.Sprintf("checking shared %v policy expected to be succeed: %#v", pr, err))
		repocall.Unset()
	}
}

func TestDeletePolicies(t *testing.T) {
	svc, _, _, prepo := newService()

	repocall := prepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(nil)
	prs := []auth.PolicyReq{{Object: "obj", ObjectType: "object", Relation: "rel", Subject: "sub", SubjectType: "subject"}}
	err := svc.DeletePolicies(context.Background(), prs)
	require.Nil(t, err, fmt.Sprintf("adding %v policies expected to succeed: %v", prs, err))
	repocall.Unset()
}

func TestListPolicies(t *testing.T) {
	svc, _, _, prepo := newService()

	pageLen := 15

	// Add arbitrary policies to the user.
	var prs []auth.PolicyReq
	for i := 0; i < pageLen; i++ {
		prs = append(prs, auth.PolicyReq{
			Subject:     id,
			SubjectType: auth.UserType,
			Relation:    auth.ViewerRelation,
			Object:      fmt.Sprintf("thing-%d", i),
			ObjectType:  auth.ThingType,
		})
	}
	repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(nil)
	err := svc.AddPolicies(context.Background(), prs)
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))
	repocall.Unset()

	expectedPolicies := make([]auth.PolicyRes, pageLen)
	repocall2 := prepo.On("RetrieveObjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, nil)
	page, err := svc.ListObjects(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission}, "", 100)
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
	repocall2.Unset()
}
