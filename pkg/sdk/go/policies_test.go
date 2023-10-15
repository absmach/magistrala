package sdk_test

// import (
// 	"fmt"
// 	"net/http"
// 	"net/http/httptest"
// 	"testing"
// 	"time"

// 	"github.com/go-zoo/bone"
// 	"github.com/mainflux/mainflux/internal/apiutil"
// 	mflog "github.com/mainflux/mainflux/logger"
// 	"github.com/mainflux/mainflux/pkg/errors"
// 	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
// 	tclients "github.com/mainflux/mainflux/things/clients"
// 	tmocks "github.com/mainflux/mainflux/things/clients/mocks"
// 	tgmocks "github.com/mainflux/mainflux/things/groups/mocks"
// 	tpolicies "github.com/mainflux/mainflux/things/policies"
// 	tapi "github.com/mainflux/mainflux/things/policies/api/http"
// 	tpmocks "github.com/mainflux/mainflux/things/policies/mocks"
// 	uclients "github.com/mainflux/mainflux/users/clients"
// 	umocks "github.com/mainflux/mainflux/users/clients/mocks"
// 	"github.com/mainflux/mainflux/users/jwt"
// 	upolicies "github.com/mainflux/mainflux/users/policies"
// 	uapi "github.com/mainflux/mainflux/users/policies/api/http"
// 	upmocks "github.com/mainflux/mainflux/users/policies/mocks"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/mock"
// )

// var utadminPolicy = umocks.SubjectSet{Subject: "things", Relation: []string{"g_add"}}

// func newUsersPolicyServer(svc upolicies.Service) *httptest.Server {
// 	logger := mflog.NewMock()
// 	mux := bone.New()
// 	uapi.MakeHandler(svc, mux, logger)

// 	return httptest.NewServer(mux)
// }

// func newThingsPolicyServer(svc tclients.Service, psvc tpolicies.Service) *httptest.Server {
// 	logger := mflog.NewMock()
// 	mux := bone.New()
// 	tapi.MakeHandler(svc, psvc, mux, logger)

// 	return httptest.NewServer(mux)
// }

// func TestCreatePolicyUser(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	clientPolicy := sdk.Policy{Object: object, Actions: []string{"m_write", "g_add"}, Subject: subject}

// 	cases := []struct {
// 		desc   string
// 		policy sdk.Policy
// 		page   sdk.PolicyPage
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc: "add new policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "add existing policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{Policies: []sdk.Policy{clientPolicy}},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedCreation), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "add a new policy with owner",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				OwnerID: generateUUID(t),
// 				Object:  "objwithowner",
// 				Actions: []string{"m_read"},
// 				Subject: "subwithowner",
// 			},
// 			err:   nil,
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with more actions",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj2",
// 				Actions: []string{"c_delete", "c_update", "c_list"},
// 				Subject: "sub2",
// 			},
// 			err:   nil,
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with wrong action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj3",
// 				Actions: []string{"wrong"},
// 				Subject: "sub3",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty object",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Subject: "sub4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyObj), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty subject",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Object:  "obj4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicySub), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Subject: "sub5",
// 				Object:  "obj5",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicyAct, http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := pRepo.On("Save", mock.Anything, mock.Anything).Return(tc.err)
// 		err := mfsdk.CreateUserPolicy(tc.policy, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestAuthorizeUser(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	cases := []struct {
// 		desc   string
// 		policy sdk.AccessRequest
// 		page   sdk.PolicyPage
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc: "authorize a valid policy with client entity",
// 			policy: sdk.AccessRequest{
// 				Subject:    subject,
// 				Object:     object,
// 				Action:     "c_list",
// 				EntityType: "client",
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "authorize a valid policy with group entity",
// 			policy: sdk.AccessRequest{
// 				Subject:    subject,
// 				Object:     object,
// 				Action:     "g_add",
// 				EntityType: "group",
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "authorize a policy with wrong action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.AccessRequest{
// 				Object:     "obj3",
// 				Action:     "wrong",
// 				Subject:    "sub3",
// 				EntityType: "client",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "authorize a policy with empty object",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.AccessRequest{
// 				Action:     "c_delete",
// 				Subject:    "sub4",
// 				EntityType: "client",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyObj), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "authorize a policy with empty subject",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.AccessRequest{
// 				Action:     "c_delete",
// 				Object:     "obj4",
// 				EntityType: "client",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicySub), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "authorize a policy with empty action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.AccessRequest{
// 				Subject:    "sub5",
// 				Object:     "obj5",
// 				EntityType: "client",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		ok, err := mfsdk.AuthorizeUser(tc.policy, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			assert.True(t, ok, fmt.Sprintf("%s: expected true, got false", tc.desc))
// 			ok := repoCall.Parent.AssertCalled(t, "CheckAdmin", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("CheckAdmin was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestAssign(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()
// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	clientPolicy := sdk.Policy{Object: object, Actions: []string{"m_write", "g_add"}, Subject: subject}

// 	cases := []struct {
// 		desc   string
// 		policy sdk.Policy
// 		page   sdk.PolicyPage
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc: "add new policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 		},
// 		{
// 			desc: "add existing policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{Policies: []sdk.Policy{clientPolicy}},
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedCreation), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "add a new policy with owner",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				OwnerID: generateUUID(t),
// 				Object:  "objwithowner",
// 				Actions: []string{"m_read"},
// 				Subject: "subwithowner",
// 			},
// 			err:   nil,
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with more actions",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj2",
// 				Actions: []string{"c_delete", "c_update", "c_list"},
// 				Subject: "sub2",
// 			},
// 			err:   nil,
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with wrong action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj3",
// 				Actions: []string{"wrong"},
// 				Subject: "sub3",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty object",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Subject: "sub4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicyObj), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty subject",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Object:  "obj4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingPolicySub), http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 		{
// 			desc: "add a new policy with empty action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Subject: "sub5",
// 				Object:  "obj5",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicyAct, http.StatusInternalServerError),
// 			token: generateValidToken(t, csvc, cRepo),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := pRepo.On("Save", mock.Anything, mock.Anything).Return(tc.err)
// 		err := mfsdk.Assign(tc.policy.Actions, tc.policy.Subject, tc.policy.Object, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall1.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestUpdatePolicy(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	policy := sdk.Policy{
// 		Subject: subject,
// 		Object:  object,
// 		Actions: []string{"m_write", "g_add"},
// 	}

// 	cases := []struct {
// 		desc   string
// 		action []string
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc:   "update policy actions with valid token",
// 			action: []string{"m_write", "m_read", "g_add"},
// 			token:  generateValidToken(t, csvc, cRepo),
// 			err:    nil,
// 		},
// 		{
// 			desc:   "update policy action with invalid token",
// 			action: []string{"m_write"},
// 			token:  "non-existent",
// 			err:    errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 		},
// 		{
// 			desc:   "update policy action with wrong policy action",
// 			action: []string{"wrong"},
// 			token:  generateValidToken(t, csvc, cRepo),
// 			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 		},
// 	}

// 	for _, tc := range cases {
// 		policy.Actions = tc.action
// 		policy.CreatedAt = time.Now()
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := pRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(upolicies.PolicyPage{}, nil)
// 		repoCall2 := pRepo.On("Update", mock.Anything, mock.Anything).Return(tc.err)
// 		err := mfsdk.UpdateUserPolicy(policy, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		ok := repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
// 		assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 		repoCall2.Unset()
// 	}
// }

// func TestUpdateThingsPolicy(t *testing.T) {
// 	cRepo := new(tmocks.Repository)
// 	gRepo := new(tgmocks.Repository)
// 	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {utadminPolicy}})
// 	thingCache := tmocks.NewCache()
// 	policiesCache := tpmocks.NewCache()

// 	pRepo := new(tpmocks.Repository)
// 	psvc := tpolicies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := tclients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsPolicyServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	policy := sdk.Policy{
// 		Subject: subject,
// 		Object:  object,
// 		Actions: []string{"m_write", "g_add"},
// 	}

// 	cases := []struct {
// 		desc   string
// 		action []string
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc:   "update policy actions with valid token",
// 			action: []string{"m_write", "m_read"},
// 			token:  adminToken,
// 			err:    nil,
// 		},
// 		{
// 			desc:   "update policy action with invalid token",
// 			action: []string{"m_write"},
// 			token:  "non-existent",
// 			err:    errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized),
// 		},
// 		{
// 			desc:   "update policy action with wrong policy action",
// 			action: []string{"wrong"},
// 			token:  adminToken,
// 			err:    errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicyAct, http.StatusInternalServerError),
// 		},
// 	}

// 	for _, tc := range cases {
// 		policy.Actions = tc.action
// 		policy.CreatedAt = time.Now()
// 		repoCall := pRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tpolicies.PolicyPage{}, nil)
// 		repoCall1 := pRepo.On("Update", mock.Anything, mock.Anything).Return(tpolicies.Policy{}, tc.err)
// 		err := mfsdk.UpdateThingPolicy(policy, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		ok := repoCall.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
// 		assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestListPolicies(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)
// 	id := generateUUID(t)

// 	nPolicy := uint64(10)
// 	aPolicies := []sdk.Policy{}
// 	for i := uint64(0); i < nPolicy; i++ {
// 		pr := sdk.Policy{
// 			OwnerID: id,
// 			Actions: []string{"m_read"},
// 			Subject: fmt.Sprintf("thing_%d", i),
// 			Object:  fmt.Sprintf("client_%d", i),
// 		}
// 		if i%3 == 0 {
// 			pr.Actions = []string{"m_write"}
// 		}
// 		aPolicies = append(aPolicies, pr)
// 	}

// 	cases := []struct {
// 		desc     string
// 		token    string
// 		page     sdk.PageMetadata
// 		response []sdk.Policy
// 		err      errors.SDKError
// 	}{
// 		{
// 			desc:     "list policies with authorized token",
// 			token:    generateValidToken(t, csvc, cRepo),
// 			err:      nil,
// 			response: aPolicies,
// 		},
// 		{
// 			desc:     "list policies with invalid token",
// 			token:    invalidToken,
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized),
// 			response: []sdk.Policy(nil),
// 		},
// 		{
// 			desc:  "list policies with offset and limit",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with given name",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with given identifier",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with given ownerID",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with given subject",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with given object",
// 			token: generateValidToken(t, csvc, cRepo),
// 			err:   nil,
// 			page: sdk.PageMetadata{
// 				Offset: 6,
// 				Limit:  nPolicy,
// 			},
// 			response: aPolicies[6:10],
// 		},
// 		{
// 			desc:  "list policies with wrong action",
// 			token: generateValidToken(t, csvc, cRepo),
// 			page: sdk.PageMetadata{
// 				Action: "wrong",
// 			},
// 			response: []sdk.Policy(nil),
// 			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 		repoCall1 := pRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertUserPolicyPage(sdk.PolicyPage{Policies: tc.response}), tc.err)
// 		pp, err := mfsdk.ListUserPolicies(tc.page, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		assert.Equal(t, tc.response, pp.Policies, fmt.Sprintf("%s: expected %v, got %v", tc.desc, tc.response, pp))
// 		ok := repoCall.Parent.AssertCalled(t, "RetrieveAll", mock.Anything, mock.Anything)
// 		assert.True(t, ok, fmt.Sprintf("RetrieveAll was not called on %s", tc.desc))
// 		repoCall.Unset()
// 		repoCall1.Unset()
// 	}
// }

// func TestDeletePolicy(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	sub := generateUUID(t)
// 	pr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}
// 	cpr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}

// 	repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 := pRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertUserPolicyPage(sdk.PolicyPage{Policies: []sdk.Policy{cpr}}), nil)
// 	repoCall2 := pRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
// 	err := mfsdk.DeleteUserPolicy(pr, generateValidToken(t, csvc, cRepo))
// 	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
// 	ok := repoCall1.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on valid policy")
// 	repoCall2.Unset()
// 	repoCall1.Unset()
// 	repoCall.Unset()

// 	repoCall = pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 = pRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertUserPolicyPage(sdk.PolicyPage{Policies: []sdk.Policy{cpr}}), nil)
// 	repoCall2 = pRepo.On("Delete", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	err = mfsdk.DeleteUserPolicy(pr, invalidToken)
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized), fmt.Sprintf("expected %v got %s", pr, err))
// 	ok = repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on invalid policy")
// 	repoCall2.Unset()
// 	repoCall1.Unset()
// 	repoCall.Unset()
// }

// func TestUnassign(t *testing.T) {
// 	cRepo := new(umocks.Repository)
// 	pRepo := new(upmocks.Repository)
// 	tokenizer := jwt.NewRepository([]byte(secret), accessDuration, refreshDuration)

// 	csvc := uclients.NewService(cRepo, pRepo, tokenizer, emailer, phasher, idProvider, passRegex)
// 	svc := upolicies.NewService(pRepo, tokenizer, idProvider)
// 	ts := newUsersPolicyServer(svc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		UsersURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	sub := generateUUID(t)
// 	pr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}
// 	cpr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}

// 	repoCall := pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 := pRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertUserPolicyPage(sdk.PolicyPage{Policies: []sdk.Policy{cpr}}), nil)
// 	repoCall2 := pRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
// 	err := mfsdk.Unassign(pr.Subject, pr.Object, generateValidToken(t, csvc, cRepo))
// 	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
// 	ok := repoCall1.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on valid policy")
// 	repoCall2.Unset()
// 	repoCall1.Unset()
// 	repoCall.Unset()

// 	repoCall = pRepo.On("CheckAdmin", mock.Anything, mock.Anything).Return(nil)
// 	repoCall1 = pRepo.On("RetrieveAll", mock.Anything, mock.Anything).Return(convertUserPolicyPage(sdk.PolicyPage{Policies: []sdk.Policy{cpr}}), nil)
// 	repoCall2 = pRepo.On("Delete", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	err = mfsdk.Unassign(pr.Subject, pr.Object, invalidToken)
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthentication, sdk.ErrInvalidJWT), http.StatusUnauthorized), fmt.Sprintf("expected %v got %s", pr, err))
// 	ok = repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on invalid policy")
// 	repoCall2.Unset()
// 	repoCall1.Unset()
// 	repoCall.Unset()
// }

// func TestConnect(t *testing.T) {
// 	cRepo := new(tmocks.Repository)
// 	gRepo := new(tgmocks.Repository)
// 	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {utadminPolicy}})
// 	thingCache := tmocks.NewCache()
// 	policiesCache := tpmocks.NewCache()

// 	pRepo := new(tpmocks.Repository)
// 	psvc := tpolicies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := tclients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsPolicyServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	clientPolicy := sdk.Policy{Object: object, Actions: []string{"m_write", "g_add"}, Subject: subject}

// 	cases := []struct {
// 		desc   string
// 		policy sdk.Policy
// 		page   sdk.PolicyPage
// 		token  string
// 		err    errors.SDKError
// 		tcerr  errors.SDKError
// 	}{
// 		{
// 			desc: "add new policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: adminToken,
// 			err:   nil,
// 			tcerr: nil,
// 		},
// 		{
// 			desc: "add existing policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{Policies: []sdk.Policy{clientPolicy}},
// 			token: adminToken,
// 			err:   errors.NewSDKErrorWithStatus(sdk.ErrFailedCreation, http.StatusInternalServerError),
// 			tcerr: errors.NewSDKError(sdk.ErrFailedCreation),
// 		},
// 		{
// 			desc: "add a new policy with owner",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				OwnerID: generateUUID(t),
// 				Object:  "objwithowner",
// 				Actions: []string{"m_read"},
// 				Subject: "subwithowner",
// 			},
// 			err:   nil,
// 			tcerr: nil,
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with more actions",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj2",
// 				Actions: []string{"c_delete", "c_update", "c_list"},
// 				Subject: "sub2",
// 			},
// 			err:   nil,
// 			tcerr: nil,
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with wrong action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj3",
// 				Actions: []string{"wrong"},
// 				Subject: "sub3",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicyAct, http.StatusInternalServerError),
// 			tcerr: errors.NewSDKError(apiutil.ErrMalformedPolicyAct),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty object",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Subject: "sub4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
// 			tcerr: errors.NewSDKError(apiutil.ErrMissingID),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty subject",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Object:  "obj4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
// 			tcerr: errors.NewSDKError(apiutil.ErrMissingID),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Subject: "sub5",
// 				Object:  "obj5",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(apiutil.ErrMalformedPolicyAct, http.StatusInternalServerError),
// 			tcerr: errors.NewSDKError(apiutil.ErrMalformedPolicyAct),
// 			token: adminToken,
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("Save", mock.Anything, mock.Anything).Return(convertThingPolicy(tc.policy), tc.tcerr)
// 		conn := sdk.ConnectionIDs{ChannelIDs: []string{tc.policy.Object}, ThingIDs: []string{tc.policy.Subject}, Actions: tc.policy.Actions}
// 		err := mfsdk.Connect(conn, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestConnectThing(t *testing.T) {
// 	cRepo := new(tmocks.Repository)
// 	gRepo := new(tgmocks.Repository)
// 	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {utadminPolicy}})
// 	thingCache := tmocks.NewCache()
// 	policiesCache := tpmocks.NewCache()

// 	pRepo := new(tpmocks.Repository)
// 	psvc := tpolicies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := tclients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsPolicyServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	clientPolicy := sdk.Policy{Object: object, Actions: []string{"m_write", "g_add"}, Subject: subject}

// 	cases := []struct {
// 		desc   string
// 		policy sdk.Policy
// 		page   sdk.PolicyPage
// 		token  string
// 		err    errors.SDKError
// 	}{
// 		{
// 			desc: "add new policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{},
// 			token: adminToken,
// 			err:   nil,
// 		},
// 		{
// 			desc: "add existing policy",
// 			policy: sdk.Policy{
// 				Subject: subject,
// 				Object:  object,
// 				Actions: []string{"m_write", "g_add"},
// 			},
// 			page:  sdk.PolicyPage{Policies: []sdk.Policy{clientPolicy}},
// 			token: adminToken,
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, sdk.ErrFailedCreation), http.StatusInternalServerError),
// 		},
// 		{
// 			desc: "add a new policy with owner",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				OwnerID: generateUUID(t),
// 				Object:  "objwithowner",
// 				Actions: []string{"m_read"},
// 				Subject: "subwithowner",
// 			},
// 			err:   nil,
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with more actions",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj2",
// 				Actions: []string{"c_delete", "c_update", "c_list"},
// 				Subject: "sub2",
// 			},
// 			err:   nil,
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with wrong action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Object:  "obj3",
// 				Actions: []string{"wrong"},
// 				Subject: "sub3",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty object",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Subject: "sub4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty subject",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Actions: []string{"c_delete"},
// 				Object:  "obj4",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
// 			token: adminToken,
// 		},
// 		{
// 			desc: "add a new policy with empty action",
// 			page: sdk.PolicyPage{},
// 			policy: sdk.Policy{
// 				Subject: "sub5",
// 				Object:  "obj5",
// 			},
// 			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMalformedPolicyAct), http.StatusInternalServerError),
// 			token: adminToken,
// 		},
// 	}

// 	for _, tc := range cases {
// 		repoCall := pRepo.On("Save", mock.Anything, mock.Anything).Return(convertThingPolicy(tc.policy), tc.err)
// 		err := mfsdk.ConnectThing(tc.policy.Subject, tc.policy.Object, tc.token)
// 		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
// 		if tc.err == nil {
// 			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
// 			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
// 		}
// 		repoCall.Unset()
// 	}
// }

// func TestDisconnectThing(t *testing.T) {
// 	cRepo := new(tmocks.Repository)
// 	gRepo := new(tgmocks.Repository)
// 	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {utadminPolicy}})
// 	thingCache := tmocks.NewCache()
// 	policiesCache := tpmocks.NewCache()

// 	pRepo := new(tpmocks.Repository)
// 	psvc := tpolicies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := tclients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsPolicyServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	sub := generateUUID(t)
// 	pr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}

// 	repoCall := pRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
// 	err := mfsdk.DisconnectThing(pr.Subject, pr.Object, adminToken)
// 	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
// 	ok := repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on valid policy")
// 	repoCall.Unset()

// 	repoCall = pRepo.On("Delete", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	err = mfsdk.DisconnectThing(pr.Subject, pr.Object, invalidToken)
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized), fmt.Sprintf("expected %v got %s", pr, err))
// 	ok = repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on invalid policy")
// 	repoCall.Unset()
// }

// func TestDisconnect(t *testing.T) {
// 	cRepo := new(tmocks.Repository)
// 	gRepo := new(tgmocks.Repository)
// 	uauth := umocks.NewAuthService(users, map[string][]umocks.SubjectSet{adminID: {utadminPolicy}})
// 	thingCache := tmocks.NewCache()
// 	policiesCache := tpmocks.NewCache()

// 	pRepo := new(tpmocks.Repository)
// 	psvc := tpolicies.NewService(uauth, pRepo, policiesCache, idProvider)

// 	svc := tclients.NewService(uauth, psvc, cRepo, gRepo, thingCache, idProvider)
// 	ts := newThingsPolicyServer(svc, psvc)
// 	defer ts.Close()

// 	conf := sdk.Config{
// 		ThingsURL: ts.URL,
// 	}
// 	mfsdk := sdk.NewSDK(conf)

// 	sub := generateUUID(t)
// 	pr := sdk.Policy{Object: authoritiesObj, Actions: []string{"m_read", "g_add", "c_delete"}, Subject: sub}

// 	repoCall := pRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
// 	conn := sdk.ConnectionIDs{ChannelIDs: []string{pr.Object}, ThingIDs: []string{pr.Subject}}
// 	err := mfsdk.Disconnect(conn, adminToken)
// 	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
// 	ok := repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on valid policy")
// 	repoCall.Unset()

// 	repoCall = pRepo.On("Delete", mock.Anything, mock.Anything).Return(sdk.ErrFailedRemoval)
// 	conn = sdk.ConnectionIDs{ChannelIDs: []string{pr.Object}, ThingIDs: []string{pr.Subject}}
// 	err = mfsdk.Disconnect(conn, invalidToken)
// 	assert.Equal(t, err, errors.NewSDKErrorWithStatus(errors.Wrap(errors.ErrAuthorization, errors.ErrAuthentication), http.StatusUnauthorized), fmt.Sprintf("expected %v got %s", pr, err))
// 	ok = repoCall.Parent.AssertCalled(t, "Delete", mock.Anything, mock.Anything)
// 	assert.True(t, ok, "Delete was not called on invalid policy")
// 	repoCall.Unset()
// }
