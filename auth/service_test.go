// Copyright (c) Magistrala
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

var idProvider = uuid.New()

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

	readPolicy   = "read"
	writePolicy  = "write"
	deletePolicy = "delete"
)

func newService() (auth.Service, *mocks.Keys) {
	krepo := new(mocks.Keys)
	prepo := new(mocks.PolicyAgent)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))

	return auth.New(krepo, idProvider, t, prepo, loginDuration, refreshDuration), krepo
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

	pr := auth.PolicyReq{Object: "obj", Relation: "rel", Subject: "sub"}
	err := svc.AddPolicy(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("adding %v policy expected to succeed: %v", pr, err))

	err = svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("checking shared %v policy expected to be succeed: %#v", pr, err))
}

func TestDeletePolicy(t *testing.T) {
	svc, _ := newService()

	pr := auth.PolicyReq{Object: authoritiesObj, Relation: memberRelation, Subject: id}
	err := svc.DeletePolicy(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("deleting %v policy expected to succeed: %s", pr, err))
}

func TestAddPolicies(t *testing.T) {
	svc, _ := newService()
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  id,
		IssuedAt: time.Now(),
	}

	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	thingID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tmpID := "tmpid"

	// Add read policy to users.
	err = svc.AddPolicies(context.Background(), apiToken.AccessToken, thingID, []string{id, tmpID}, []string{readPolicy})
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))

	// Add write and delete policies to users.
	err = svc.AddPolicies(context.Background(), apiToken.AccessToken, thingID, []string{id, tmpID}, []string{writePolicy, deletePolicy})
	assert.Nil(t, err, fmt.Sprintf("adding multiple policies expected to succeed: %s", err))

	cases := []struct {
		desc   string
		policy auth.PolicyReq
		err    error
	}{
		{
			desc:   "check valid 'read' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: readPolicy, Subject: id},
			err:    nil,
		},
		{
			desc:   "check valid 'write' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: writePolicy, Subject: id},
			err:    nil,
		},
		{
			desc:   "check valid 'delete' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: deletePolicy, Subject: id},
			err:    nil,
		},
		{
			desc:   "check valid 'read' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: readPolicy, Subject: tmpID},
			err:    nil,
		},
		{
			desc:   "check valid 'write' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: writePolicy, Subject: tmpID},
			err:    nil,
		},
		{
			desc:   "check valid 'delete' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: deletePolicy, Subject: tmpID},
			err:    nil,
		},
		{
			desc:   "check invalid 'access' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: "access", Subject: id},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check invalid 'access' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: "access", Subject: tmpID},
			err:    errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.Authorize(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.err, err))
	}
}

func TestDeletePolicies(t *testing.T) {
	svc, _ := newService()
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  id,
		IssuedAt: time.Now(),
	}

	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	thingID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tmpID := "tmpid"
	memberPolicy := "member"

	// Add read, write and delete policies to users.
	err = svc.AddPolicies(context.Background(), apiToken.AccessToken, thingID, []string{id, tmpID}, []string{readPolicy, writePolicy, deletePolicy, memberPolicy})
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))

	// Delete multiple policies from single user.
	err = svc.DeletePolicies(context.Background(), apiToken.AccessToken, thingID, []string{id}, []string{readPolicy, writePolicy})
	assert.Nil(t, err, fmt.Sprintf("deleting policies from single user expected to succeed: %s", err))

	// Delete multiple policies from multiple user.
	err = svc.DeletePolicies(context.Background(), apiToken.AccessToken, thingID, []string{id, tmpID}, []string{deletePolicy, memberPolicy})
	assert.Nil(t, err, fmt.Sprintf("deleting policies from multiple user expected to succeed: %s", err))

	cases := []struct {
		desc   string
		policy auth.PolicyReq
		err    error
	}{
		{
			desc:   "check non-existing 'read' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: readPolicy, Subject: id},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check non-existing 'write' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: writePolicy, Subject: id},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check non-existing 'delete' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: deletePolicy, Subject: id},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check non-existing 'member' policy of user with id",
			policy: auth.PolicyReq{Object: thingID, Relation: memberPolicy, Subject: id},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check non-existing 'delete' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: deletePolicy, Subject: tmpID},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check non-existing 'member' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: memberPolicy, Subject: tmpID},
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "check valid 'read' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: readPolicy, Subject: tmpID},
			err:    nil,
		},
		{
			desc:   "check valid 'write' policy of user with tmpid",
			policy: auth.PolicyReq{Object: thingID, Relation: writePolicy, Subject: tmpID},
			err:    nil,
		},
	}

	for _, tc := range cases {
		err := svc.Authorize(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.err, err))
	}
}

func TestListPolicies(t *testing.T) {
	svc, _ := newService()
	secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		Subject:  id,
		IssuedAt: time.Now(),
	}

	apiToken, err := svc.Issue(context.Background(), secret.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))
	pageLen := 15

	// Add arbitrary policies to the user.
	for i := 0; i < pageLen; i++ {
		err = svc.AddPolicies(context.Background(), apiToken.AccessToken, fmt.Sprintf("thing-%d", i), []string{id}, []string{readPolicy})
		assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))
	}

	page, err := svc.ListObjects(context.Background(), auth.PolicyReq{Subject: id, Relation: readPolicy}, "", 100)
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
}
