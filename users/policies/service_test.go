// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/internal/testsutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/users/clients"
	"github.com/mainflux/mainflux/users/clients/mocks"
	"github.com/mainflux/mainflux/users/hasher"
	"github.com/mainflux/mainflux/users/jwt"
	"github.com/mainflux/mainflux/users/policies"
	pmocks "github.com/mainflux/mainflux/users/policies/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider      = uuid.New()
	phasher         = hasher.New()
	secret          = "strongsecret"
	inValidToken    = "invalidToken"
	memberActions   = []string{"g_list"}
	authoritiesObj  = "authorities"
	passRegex       = regexp.MustCompile("^.{8,}$")
	accessDuration  = time.Minute * 1
	refreshDuration = time.Minute * 10
)

func TestAddPolicy(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := policies.NewService(pRepo, tokenizer, idProvider)

	policy := policies.Policy{Object: testsutil.GenerateUUID(t, idProvider), Subject: testsutil.GenerateUUID(t, idProvider), Actions: []string{"c_list"}}

	cases := []struct {
		desc   string
		policy policies.Policy
		token  string
		err    error
	}{
		{
			desc:   "add new policy",
			policy: policy,
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:    nil,
		},
		{
			desc: "add a new policy with owner",
			policy: policies.Policy{
				OwnerID: testsutil.GenerateUUID(t, idProvider),
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"m_read"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with c_update action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"c_update"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with c_update and c_list action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"c_update", "c_list"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with g_update action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"g_update"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with g_update and g_list action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"g_update", "g_list"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with more actions",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"c_delete", "c_update", "c_list"},
			},
			err:   nil,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with wrong action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"wrong"},
			},
			err:   apiutil.ErrMalformedPolicyAct,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with empty object",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"c_delete"},
			},
			err:   apiutil.ErrMissingPolicyObj,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with empty subject",
			policy: policies.Policy{
				Object:  testsutil.GenerateUUID(t, idProvider),
				Actions: []string{"c_delete"},
			},
			err:   apiutil.ErrMissingPolicySub,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
		{
			desc: "add a new policy with empty action",
			policy: policies.Policy{
				Subject: testsutil.GenerateUUID(t, idProvider),
				Object:  testsutil.GenerateUUID(t, idProvider),
			},
			err:   apiutil.ErrMalformedPolicyAct,
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(errors.ErrAuthorization)
		repoCall1 := pRepo.On("EvaluateGroupAccess", context.Background(), mock.Anything).Return(policies.Policy{}, tc.err)
		repoCall2 := pRepo.On("EvaluateUserAccess", context.Background(), mock.Anything).Return(policies.Policy{}, tc.err)
		repoCall3 := pRepo.On("Save", context.Background(), mock.Anything).Return(tc.err)
		err := svc.AddPolicy(context.Background(), tc.token, tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.policy.Subject = tc.token
			aReq := policies.AccessRequest{Subject: tc.policy.Subject, Object: tc.policy.Object, Action: tc.policy.Actions[0], Entity: "client"}
			err = svc.Authorize(context.Background(), aReq)
			require.Nil(t, err, fmt.Sprintf("checking shared %v policy expected to be succeed: %#v", tc.policy, err))
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
			ok = repoCall3.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := policies.NewService(pRepo, tokenizer, idProvider)

	cases := []struct {
		desc   string
		policy policies.AccessRequest
		err    error
	}{
		{
			desc: "check valid policy in client domain",
			policy: policies.AccessRequest{
				Object:  testsutil.GenerateUUID(t, idProvider),
				Action:  "c_update",
				Subject: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
				Entity:  "client",
			},
			err: nil,
		},
		{
			desc: "check valid policy in group domain",
			policy: policies.AccessRequest{
				Object:  testsutil.GenerateUUID(t, idProvider),
				Action:  "g_update",
				Subject: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
				Entity:  "group",
			},
			err: errors.ErrConflict,
		},
		{
			desc: "check invalid policy in client domain",
			policy: policies.AccessRequest{
				Object:  testsutil.GenerateUUID(t, idProvider),
				Action:  "c_update",
				Subject: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
				Entity:  "client",
			},
			err: nil,
		},
		{
			desc: "check invalid policy in group domain",
			policy: policies.AccessRequest{
				Object:  testsutil.GenerateUUID(t, idProvider),
				Action:  "g_update",
				Subject: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
				Entity:  "group",
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(tc.err)
		repoCall1 := &mock.Call{}
		switch tc.policy.Entity {
		case "client":
			repoCall1 = pRepo.On("EvaluateUserAccess", context.Background(), mock.Anything).Return(policies.Policy{}, tc.err)
		case "group":
			repoCall1 = pRepo.On("EvaluateGroupAccess", context.Background(), mock.Anything).Return(policies.Policy{}, tc.err)
		}
		err := svc.Authorize(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestDeletePolicy(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := policies.NewService(pRepo, tokenizer, idProvider)

	pr := policies.Policy{Object: authoritiesObj, Actions: memberActions, Subject: testsutil.GenerateUUID(t, idProvider)}

	repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
	repoCall1 := pRepo.On("Delete", context.Background(), pr).Return(nil)
	err := svc.DeletePolicy(context.Background(), testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher), pr)
	require.Nil(t, err, fmt.Sprintf("deleting %v policy expected to succeed: %s", pr, err))
	ok := repoCall.Parent.AssertCalled(t, "Delete", context.Background(), pr)
	assert.True(t, ok, "Delete was not called on deleting policy")
	repoCall.Unset()
	repoCall1.Unset()
}

func TestListPolicies(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := policies.NewService(pRepo, tokenizer, idProvider)

	id := testsutil.GenerateUUID(t, idProvider)

	readPolicy := "m_read"
	writePolicy := "m_write"

	nPolicy := uint64(10)
	aPolicies := []policies.Policy{}
	for i := uint64(0); i < nPolicy; i++ {
		pr := policies.Policy{
			OwnerID: id,
			Actions: []string{readPolicy},
			Subject: fmt.Sprintf("thing_%d", i),
			Object:  fmt.Sprintf("client_%d", i),
		}
		if i%3 == 0 {
			pr.Actions = []string{writePolicy}
		}
		aPolicies = append(aPolicies, pr)
	}

	cases := []struct {
		desc     string
		token    string
		page     policies.Page
		response policies.PolicyPage
		err      error
	}{
		{
			desc:  "list policies with authorized token",
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:   nil,
			response: policies.PolicyPage{
				Page: policies.Page{
					Offset: 0,
					Total:  nPolicy,
				},
				Policies: aPolicies,
			},
		},
		{
			desc:  "list policies with invalid token",
			token: inValidToken,
			err:   errors.ErrAuthentication,
			response: policies.PolicyPage{
				Page: policies.Page{
					Offset: 0,
				},
			},
		},
		{
			desc:  "list policies with offset and limit",
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			page: policies.Page{
				Offset: 6,
				Limit:  nPolicy,
			},
			response: policies.PolicyPage{
				Page: policies.Page{
					Offset: 6,
					Total:  nPolicy,
				},
				Policies: aPolicies[6:10],
			},
		},
		{
			desc:  "list policies with wrong action",
			token: testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			page: policies.Page{
				Action: "wrong",
			},
			response: policies.PolicyPage{},
			err:      apiutil.ErrMalformedPolicyAct,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := pRepo.On("RetrieveAll", context.Background(), tc.page).Return(tc.response, tc.err)
		page, err := svc.ListPolicies(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected size %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall.Parent.AssertCalled(t, "RetrieveAll", context.Background(), tc.page)
			assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
		}
		repoCall1.Unset()
		repoCall.Unset()
	}
}

func TestUpdatePolicies(t *testing.T) {
	cRepo := new(mocks.Repository)
	pRepo := new(pmocks.Repository)
	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)
	e := mocks.NewEmailer()
	csvc := clients.NewService(cRepo, pRepo, tokenizer, e, phasher, idProvider, passRegex)
	svc := policies.NewService(pRepo, tokenizer, idProvider)

	policy := policies.Policy{Object: "obj1", Actions: []string{"m_read"}, Subject: "sub1"}

	cases := []struct {
		desc   string
		action []string
		token  string
		err    error
	}{
		{
			desc:   "update policy actions with valid token",
			action: []string{"m_write"},
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:    nil,
		},
		{
			desc:   "update policy action with invalid token",
			action: []string{"m_write"},
			token:  "non-existent",
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update policy action with wrong policy action",
			action: []string{"wrong"},
			token:  testsutil.GenerateValidToken(t, testsutil.GenerateUUID(t, idProvider), csvc, cRepo, phasher),
			err:    apiutil.ErrMalformedPolicyAct,
		},
	}

	for _, tc := range cases {
		policy.Actions = tc.action
		repoCall := pRepo.On("CheckAdmin", context.Background(), mock.Anything).Return(nil)
		repoCall1 := pRepo.On("RetrieveAll", context.Background(), mock.Anything).Return(policies.PolicyPage{Policies: []policies.Policy{policy}}, nil)
		repoCall2 := pRepo.On("Update", context.Background(), mock.Anything).Return(tc.err)
		err := svc.UpdatePolicy(context.Background(), tc.token, policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", context.Background(), mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}
