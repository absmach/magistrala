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
	accessToken     = "access"
)

func newService() (auth.Service, *mocks.Keys) {
	krepo := new(mocks.Keys)
	prepo := new(mocks.PolicyAgent)
	drepo := new(mocks.DomainsRepo)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))

	return auth.New(krepo, drepo, idProvider, t, prepo, loginDuration, refreshDuration), krepo
}

func TestIssue(t *testing.T) {
	svc, krepo := newService()

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
			desc: "issue login key with no time",
			key: auth.Key{
				Type: auth.AccessKey,
			},
			token: accessToken,
			err:   auth.ErrInvalidKeyIssuedAt,
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
			err:   errors.ErrAuthentication,
		},
		{
			desc: "issue API key with no time",
			key: auth.Key{
				Type: auth.APIKey,
			},
			token: accessToken,
			err:   auth.ErrInvalidKeyIssuedAt,
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
		{
			desc: "issue recovery with no issue time",
			key: auth.Key{
				Type: auth.RecoveryKey,
			},
			token: accessToken,
			err:   auth.ErrInvalidKeyIssuedAt,
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
	svc, _ := newService()
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		Type:     auth.APIKey,
		IssuedAt: time.Now(),
		Subject:  id,
	}
	_, err = svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

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
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	svc, _ := newService()
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  id,
		IssuedAt: time.Now(),
	}

	userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login's key expected to succeed: %s", err))

	resetToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

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
			err:   errors.ErrNotFound,
		},
		{
			desc: "retrieve with wrong login key",
			// id:    apiKey.ID,
			token: "wrong",
			err:   errors.ErrAuthentication,
		},
		{
			desc: "retrieve with API token",
			// id:    apiKey.ID,
			token: apiToken.AccessToken,
			err:   errors.ErrAuthentication,
		},
		{
			desc: "retrieve with reset token",
			// id:    apiKey.ID,
			token: resetToken.AccessToken,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc, _ := newService()

	loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	apiSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, Subject: id, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	exp1 := time.Now().Add(-2 * time.Second)
	expSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))

	invalidSecret, err := svc.Issue(context.Background(), loginSecret.AccessToken, auth.Key{Type: 22, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

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
			err:  auth.ErrAPIKeyExpired,
		},
		{
			desc: "identify expired key",
			key:  invalidSecret.AccessToken,
			idt:  "",
			err:  errors.ErrAuthentication,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  "",
			err:  errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
	}
}

func TestAuthorize(t *testing.T) {
	svc, _ := newService()

	pr := auth.PolicyReq{Object: authoritiesObj, Relation: memberRelation, Subject: id}
	err := svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("authorizing initial %v policy expected to succeed: %s", pr, err))
}

func TestAddPolicy(t *testing.T) {
	svc, _ := newService()

	prs := []auth.PolicyReq{{Object: "obj", ObjectType: "object", Relation: "rel", Subject: "sub", SubjectType: "subject"}}
	err := svc.AddPolicies(context.Background(), prs)
	require.Nil(t, err, fmt.Sprintf("adding %v policies expected to succeed: %v", prs, err))

	for _, pr := range prs {
		err = svc.Authorize(context.Background(), pr)
		require.Nil(t, err, fmt.Sprintf("checking shared %v policy expected to be succeed: %#v", pr, err))
	}
}

func TestDeletePolicies(t *testing.T) {
	svc, _ := newService()

	prs := []auth.PolicyReq{{Object: "obj", ObjectType: "object", Relation: "rel", Subject: "sub", SubjectType: "subject"}}
	err := svc.DeletePolicies(context.Background(), prs)
	require.Nil(t, err, fmt.Sprintf("adding %v policies expected to succeed: %v", prs, err))
}

func TestListPolicies(t *testing.T) {
	svc, _ := newService()

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
	err := svc.AddPolicies(context.Background(), prs)
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))

	page, err := svc.ListObjects(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission}, "", 100)
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
}
