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
	validID         = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
)

var (
	errIssueUser          = errors.New("failed to issue new login key")
	errCreateDomainPolicy = errors.New("failed to create domain policy")
	// ErrExpiry indicates that the token is expired.
	ErrExpiry    = errors.New("token is expired")
	inValidToken = "invalid"
	inValid      = "invalid"
)

func newService() (auth.Service, *mocks.KeyRepository, string, *mocks.PolicyAgent, *mocks.DomainsRepository) {
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

	return auth.New(krepo, drepo, idProvider, t, prepo, loginDuration, refreshDuration, invalidDuration), krepo, token, prepo, drepo
}

func TestIssue(t *testing.T) {
	svc, krepo, accessToken, _, _ := newService()

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
	svc, krepo, _, _, _ := newService()
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
	svc, krepo, _, _, _ := newService()
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
	svc, krepo, _, prepo, _ := newService()

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
	svc, _, _, prepo, _ := newService()

	repocall := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
	pr := auth.PolicyReq{Object: authoritiesObj, Relation: memberRelation, Subject: id}
	err := svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("authorizing initial %v policy expected to succeed: %s", pr, err))
	repocall.Unset()
}

func TestAddPolicy(t *testing.T) {
	svc, _, _, prepo, _ := newService()

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
	svc, _, _, prepo, _ := newService()

	repocall := prepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(nil)
	prs := []auth.PolicyReq{{Object: "obj", ObjectType: "object", Relation: "rel", Subject: "sub", SubjectType: "subject"}}
	err := svc.DeletePolicies(context.Background(), prs)
	require.Nil(t, err, fmt.Sprintf("adding %v policies expected to succeed: %v", prs, err))
	repocall.Unset()
}

func TestListPolicies(t *testing.T) {
	svc, _, _, prepo, _ := newService()

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

func TestListAllObjects(t *testing.T) {
	svc, _, _, prepo, _ := newService()

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
	page, err := svc.ListAllObjects(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
	repocall2.Unset()
}

func TestCountObjects(t *testing.T) {
	svc, _, _, prepo, _ := newService()

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

	repocall2 := prepo.On("CountObjects", mock.Anything, mock.Anything, mock.Anything).Return(pageLen, nil)
	count, err := svc.CountObjects(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("counting policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, count, fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, count, err))
	repocall2.Unset()
}

func TestListSubjects(t *testing.T) {
	svc, _, _, prepo, _ := newService()

	pageLen := 15

	// Add arbitrary policies to the user.
	var prs []auth.PolicyReq
	for i := 0; i < pageLen; i++ {
		prs = append(prs, auth.PolicyReq{
			Subject:     fmt.Sprintf("user-%d", i),
			SubjectType: auth.UserType,
			Relation:    auth.ViewerRelation,
			Object:      id,
			ObjectType:  auth.ThingType,
		})
	}
	repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(nil)
	err := svc.AddPolicies(context.Background(), prs)
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))
	repocall.Unset()

	expectedPolicies := make([]auth.PolicyRes, pageLen)
	repocall2 := prepo.On("RetrieveSubjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, nil)
	page, err := svc.ListSubjects(context.Background(), auth.PolicyReq{Object: id, ObjectType: auth.ThingType, Permission: auth.ViewPermission}, "", 100)
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
	repocall2.Unset()
}

func TestListAllSubjects(t *testing.T) {
	svc, _, _, prepo, _ := newService()

	pageLen := 15

	// Add arbitrary policies to the user.
	var prs []auth.PolicyReq
	for i := 0; i < pageLen; i++ {
		prs = append(prs, auth.PolicyReq{
			Subject:     fmt.Sprintf("user-%d", i),
			SubjectType: auth.UserType,
			Relation:    auth.ViewerRelation,
			Object:      id,
			ObjectType:  auth.ThingType,
		})
	}
	repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(nil)
	err := svc.AddPolicies(context.Background(), prs)
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))
	repocall.Unset()

	expectedPolicies := make([]auth.PolicyRes, pageLen)
	repocall2 := prepo.On("RetrieveSubjects", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, nil)
	page, err := svc.ListAllSubjects(context.Background(), auth.PolicyReq{Object: id, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, len(page.Policies), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page.Policies), err))
	repocall2.Unset()
}

func TestCountSubjects(t *testing.T) {
	svc, _, _, prepo, _ := newService()

	pageLen := 15

	// Add arbitrary policies to the user.
	var prs []auth.PolicyReq
	for i := 0; i < pageLen; i++ {
		prs = append(prs, auth.PolicyReq{
			Subject:     fmt.Sprintf("user-%d", i),
			SubjectType: auth.UserType,
			Relation:    auth.ViewerRelation,
			Object:      id,
			ObjectType:  auth.ThingType,
		})
	}
	repocall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(nil)
	err := svc.AddPolicies(context.Background(), prs)
	assert.Nil(t, err, fmt.Sprintf("adding policies expected to succeed: %s", err))
	repocall.Unset()

	repocall2 := prepo.On("CountSubjects", mock.Anything, mock.Anything, mock.Anything).Return(pageLen, nil)
	count, err := svc.CountSubjects(context.Background(), auth.PolicyReq{Object: id, ObjectType: auth.ThingType, Permission: auth.ViewPermission})
	assert.Nil(t, err, fmt.Sprintf("counting policies expected to succeed: %s", err))
	assert.Equal(t, pageLen, count, fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, count, err))
	repocall2.Unset()
}

func TestListPermissions(t *testing.T) {
	svc, _, _, prepo, _ := newService()

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

	//expectedPolicies := make([]auth.PolicyRes, pageLen)
	//repocall2 := prepo.On("RetrievePermissions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(expectedPolicies, mock.Anything, nil)
	// page, err := svc.ListPermissions(context.Background(), auth.PolicyReq{Subject: id, SubjectType: auth.UserType, ObjectType: auth.ThingType, Permission: auth.ViewPermission},)
	// assert.Nil(t, err, fmt.Sprintf("listing policies expected to succeed: %s", err))
	// assert.Equal(t, pageLen, len(page), fmt.Sprintf("unexpected listing page size, expected %d, got %d: %v", pageLen, len(page), err))
	// repocall2.Unset()
}

func TestCreateDomain(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc            string
		d               auth.Domain
		token           string
		userID          string
		addPolicyErr    error
		savePolicyErr   error
		saveDomainErr   error
		deleteDomainErr error
		err             error
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
	}

	for _, tc := range cases {
		repoCall := prepo.On("AddPolicies", mock.Anything, mock.Anything).Return(tc.addPolicyErr)
		repoCall1 := drepo.On("SavePolicies", mock.Anything, mock.Anything).Return(tc.savePolicyErr)
		repoCall2 := prepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(nil)
		repoCall3 := drepo.On("DeletePolicies", mock.Anything, mock.Anything).Return(tc.deleteDomainErr)
		repoCall4 := drepo.On("Save", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.saveDomainErr)
		dom, err := svc.CreateDomain(context.Background(), tc.token, tc.d)
		fmt.Println(dom)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestRetrieveDomain(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
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
			desc:     "retrieve domain with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
		{
			desc:          "retrieve non-existing domain",
			token:         accessToken,
			domainID:      inValid,
			domainRepoErr: errors.ErrNotFound,
			err:           svcerr.ErrNotFound,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.RetrieveDomain(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestRetrieveDomainPermissions(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
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
			desc:     "retrieve domain permissions with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := prepo.On("RetrievePermissions", mock.Anything, mock.Anything, mock.Anything).Return(auth.Permissions{}, nil)
		repoCall1 := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, nil)
		repoCall2 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.RetrieveDomainPermissions(context.Background(), tc.token, tc.domainID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()

	}
}

func TestUpdateDomain(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "update domain successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "update domain with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "update domain with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.UpdateDomain(context.Background(), tc.token, tc.domainID, auth.DomainReq{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestChangeDomainStatus(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "change domain status successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "change domain status with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "change domain status with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.ChangeDomainStatus(context.Background(), tc.token, tc.domainID, auth.DomainReq{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListDomains(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "list domains successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "list domains with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list domains with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.ListDomains(context.Background(), tc.token, auth.Page{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestAssignUsers(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "assign users successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "assign users with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "assign users with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		err := svc.AssignUsers(context.Background(), tc.token, tc.domainID, []string{" ", " "}, auth.AdministratorRelation)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUnassignUsers(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "unassign users successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "unassign users with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "unassign users with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		err := svc.UnassignUsers(context.Background(), tc.token, tc.domainID, []string{" ", " "}, auth.AdministratorRelation)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListUsersDomains(t *testing.T) {
	svc, _, accessToken, prepo, drepo := newService()

	cases := []struct {
		desc          string
		token         string
		domainID      string
		domainRepoErr error
		err           error
	}{
		{
			desc:     "list users domains successfully",
			token:    accessToken,
			domainID: validID,
			err:      nil,
		},
		{
			desc:     "list users domains with invalid token",
			token:    inValidToken,
			domainID: validID,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list users domains with empty domainID",
			token:    accessToken,
			domainID: "",
			err:      nil,
		},
	}

	for _, tc := range cases {
		repoCall := drepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(auth.Domain{}, tc.domainRepoErr)
		repoCall1 := prepo.On("CheckPolicy", mock.Anything, mock.Anything).Return(nil)
		_, err := svc.ListUserDomains(context.Background(), tc.token, tc.domainID, auth.Page{})
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
			response: fmt.Sprintf("%s:%s", validID, validID),
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
		ar := auth.EncodeDomainUserID(tc.domainID, validID)
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
			domainUserID: fmt.Sprintf("%s:%s", validID, validID),
			respDomainID: validID,
			respUserID:   validID,
		},
		{
			desc:         "decode domain user id with empty domainUserID",
			domainUserID: "",
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
