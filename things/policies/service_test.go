// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/internal/testsutil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things/clients/mocks"
	"github.com/mainflux/mainflux/things/policies"
	pmocks "github.com/mainflux/mainflux/things/policies/mocks"
	umocks "github.com/mainflux/mainflux/users/policies/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	idProvider        = uuid.New()
	inValidToken      = "invalidToken"
	memberActions     = []string{"g_list"}
	adminEmail        = "admin@example.com"
	token             = "token"
	adminRelationKeys = []string{"c_update", "c_list", "c_delete", "c_share"}
)

func newService(tokens map[string]string) (policies.Service, *pmocks.Repository, *umocks.Repository) {
	adminPolicy := mocks.MockSubjectSet{Object: "things", Relation: adminRelationKeys}
	auth := mocks.NewAuthService(tokens, map[string][]mocks.MockSubjectSet{adminEmail: {adminPolicy}})
	idProvider := uuid.NewMock()
	policiesCache := pmocks.NewCache()
	pRepo := new(pmocks.Repository)
	uRepo := new(umocks.Repository)

	return policies.NewService(auth, pRepo, policiesCache, idProvider), pRepo, uRepo
}

func TestAddPolicy(t *testing.T) {
	svc, pRepo, _ := newService(map[string]string{token: adminEmail})

	policy := policies.Policy{Object: "obj1", Actions: []string{"m_read"}, Subject: "sub1"}

	cases := []struct {
		desc   string
		policy policies.Policy
		token  string
		err    error
	}{
		{
			desc:   "add new policy",
			policy: policy,
			token:  token,
			err:    nil,
		},
		{
			desc: "add a new policy with owner",
			policy: policies.Policy{
				OwnerID: testsutil.GenerateUUID(t, idProvider),
				Object:  "objwithowner",
				Actions: []string{"m_read"},
				Subject: "subwithowner",
			},
			err:   nil,
			token: token,
		},
		{
			desc: "add a new policy with more actions",
			policy: policies.Policy{
				Object:  "obj2",
				Actions: []string{"c_delete", "c_update", "c_list"},
				Subject: "sub2",
			},
			err:   nil,
			token: token,
		},
		{
			desc: "add a new policy with wrong action",
			policy: policies.Policy{
				Object:  "obj3",
				Actions: []string{"wrong"},
				Subject: "sub3",
			},
			err:   apiutil.ErrMalformedPolicyAct,
			token: token,
		},
		{
			desc: "add a new policy with empty object",
			policy: policies.Policy{
				Actions: []string{"c_delete"},
				Subject: "sub4",
			},
			err:   apiutil.ErrMissingPolicyObj,
			token: token,
		},
		{
			desc: "add a new policy with empty subject",
			policy: policies.Policy{
				Actions: []string{"c_delete"},
				Object:  "obj4",
			},
			err:   apiutil.ErrMissingPolicySub,
			token: token,
		},
		{
			desc: "add a new policy with empty action",
			policy: policies.Policy{
				Subject: "sub5",
				Object:  "obj5",
			},
			err:   apiutil.ErrMalformedPolicyAct,
			token: token,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.err)
		repoCall1 := pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.err)
		repoCall2 := pRepo.On("Save", context.Background(), mock.Anything).Return(tc.policy, tc.err)
		_, err := svc.AddPolicy(context.Background(), tc.token, false, tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		if err == nil {
			tc.policy.Subject = tc.token
			areq := policies.AccessRequest{Subject: tc.policy.Subject, Object: tc.policy.Object, Action: tc.policy.Actions[0], Entity: "client"}
			_, err = svc.Authorize(context.Background(), areq)
			require.Nil(t, err, fmt.Sprintf("checking shared %v policy expected to be succeed: %#v", tc.policy, err))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Parent.AssertCalled(t, "Save", context.Background(), mock.Anything)
		repoCall2.Unset()
	}
}

func TestAuthorize(t *testing.T) {
	svc, pRepo, _ := newService(map[string]string{token: adminEmail})

	cases := []struct {
		desc   string
		policy policies.AccessRequest
		err    error
	}{
		{
			desc:   "check valid policy in client domain",
			policy: policies.AccessRequest{Object: "client1", Action: "c_update", Subject: token, Entity: "client"},
			err:    nil,
		},
		{
			desc:   "check valid policy in group domain",
			policy: policies.AccessRequest{Object: "client2", Action: "g_update", Subject: token, Entity: "group"},
			err:    nil,
		},
		{
			desc:   "check invalid policy in client domain",
			policy: policies.AccessRequest{Object: "client3", Action: "c_update", Subject: token, Entity: "client"},
			err:    nil,
		},
		{
			desc:   "check invalid policy in group domain",
			policy: policies.AccessRequest{Object: "client4", Action: "g_update", Subject: token, Entity: "group"},
			err:    nil,
		},
	}

	for _, tc := range cases {
		// policy := policies.Policy{Object: tc.policy.Object, Actions: []string{tc.policy.Action}, Subject: tc.policy.Subject}
		repoCall := &mock.Call{}
		switch tc.policy.Entity {
		case "client":
			repoCall = pRepo.On("EvaluateThingAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.err)
		case "group":
			repoCall = pRepo.On("EvaluateGroupAccess", mock.Anything, mock.Anything).Return(policies.Policy{}, tc.err)
		}
		_, err := svc.Authorize(context.Background(), tc.policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
	}
}

func TestDeletePolicy(t *testing.T) {
	svc, pRepo, _ := newService(map[string]string{token: adminEmail})

	pr := policies.Policy{Object: testsutil.GenerateUUID(t, idProvider), Actions: memberActions, Subject: testsutil.GenerateUUID(t, idProvider)}

	repoCall := pRepo.On("Delete", context.Background(), mock.Anything).Return(nil)
	repoCall1 := pRepo.On("Retrieve", context.Background(), mock.Anything).Return(policies.PolicyPage{Policies: []policies.Policy{pr}}, nil)
	err := svc.DeletePolicy(context.Background(), token, pr)
	assert.EqualError(t, err, errors.ErrAuthorization.Error(), fmt.Sprintf("deleting %v policy expected to fail: %s", pr, err))
	repoCall.Unset()
	repoCall1.Unset()
}

func TestListPolicies(t *testing.T) {
	svc, pRepo, _ := newService(map[string]string{token: adminEmail})

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
			token: token,
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
			token: token,
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
			token: token,
			page: policies.Page{
				Action: "wrong",
			},
			response: policies.PolicyPage{},
			err:      apiutil.ErrMalformedPolicyAct,
		},
	}

	for _, tc := range cases {
		repoCall := pRepo.On("Retrieve", context.Background(), mock.Anything).Return(tc.response, tc.err)
		page, err := svc.ListPolicies(context.Background(), tc.token, tc.page)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page, fmt.Sprintf("%s: expected size %v got %v\n", tc.desc, tc.response, page))
		repoCall.Unset()
	}
}

func TestUpdatePolicies(t *testing.T) {
	svc, pRepo, _ := newService(map[string]string{token: adminEmail})

	policy := policies.Policy{Object: "obj1", Actions: []string{"m_read"}, Subject: "sub1"}

	cases := []struct {
		desc   string
		action []string
		token  string
		err    error
	}{
		{
			desc:   "update policy action with invalid token",
			action: []string{"m_write"},
			token:  "non-existent",
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update policy action with wrong policy action",
			action: []string{"wrong"},
			token:  token,
			err:    apiutil.ErrMalformedPolicyAct,
		},
	}

	for _, tc := range cases {
		policy.Actions = tc.action
		repoCall := pRepo.On("Retrieve", context.Background(), mock.Anything).Return(policies.PolicyPage{Policies: []policies.Policy{policy}}, nil)
		repoCall1 := pRepo.On("Update", context.Background(), mock.Anything).Return(policies.Policy{}, tc.err)
		_, err := svc.UpdatePolicy(context.Background(), tc.token, policy)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		repoCall.Unset()
		repoCall1.Unset()
	}
}
