// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	pkgSch "github.com/absmach/magistrala/pkg/schedule"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/api"
	"github.com/absmach/magistrala/re/mocks"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/auth"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const contentType = "application/json"

var (
	namegen      = namegenerator.NewGenerator()
	domainID     = testsutil.GenerateUUID(&testing.T{})
	userID       = testsutil.GenerateUUID(&testing.T{})
	validID      = testsutil.GenerateUUID(&testing.T{})
	validToken   = "valid"
	invalidToken = "invalid"
	now          = time.Now().UTC().Truncate(time.Minute)
	future       = now.Add(1 * time.Hour)
	schedule     = pkgSch.Schedule{
		StartDateTime:   future,
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            now,
	}
	rule = re.Rule{
		ID:       validID,
		Name:     namegen.Generate(),
		DomainID: domainID,
		Schedule: schedule,
		Metadata: re.Metadata{
			"name": "test",
		},
	}
	past           = now.Add(-1 * time.Hour)
	scheduleInPast = pkgSch.Schedule{
		StartDateTime:   past,
		Recurring:       pkgSch.Daily,
		RecurringPeriod: 1,
		Time:            past,
	}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	req.Header.Set("Referer", "http://localhost")

	return tr.client.Do(req)
}

func newRuleEngineServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	svc := new(mocks.Service)
	authn := new(authnmocks.Authentication)

	logger := smqlog.NewMock()
	mux := chi.NewRouter()
	am := smqauthn.NewAuthNMiddleware(authn, smqauthn.WithAllowUnverifiedUser(true))

	api.MakeHandler(svc, am, mux, logger, "")

	return httptest.NewServer(mux), svc, authn
}

func toJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestAddRuleEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	ruleInPast := rule
	ruleInPast.Schedule = scheduleInPast

	cases := []struct {
		desc        string
		rule        re.Rule
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcRes      re.Rule
		svcErr      error
		err         error
		len         int
	}{
		{
			desc:        "add rule successfully",
			rule:        rule,
			token:       validToken,
			contentType: contentType,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			status:      http.StatusCreated,
			svcRes:      rule,
		},
		{
			desc:        "add rule with invalid token",
			rule:        rule,
			token:       invalidToken,
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "add rule with empty token",
			token:       "",
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			rule:        rule,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:  "add rule with name that is too long",
			token: validToken,
			rule: re.Rule{
				ID:   validID,
				Name: strings.Repeat("a", 1025),
				Logic: re.Script{
					Type:  re.ScriptType(0),
					Value: "return `test` end",
				},
			},
			domainID:    domainID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "add rule with empty domainID",
			token:       validToken,
			rule:        rule,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "add rule with invalid content type",
			token:       validToken,
			domainID:    domainID,
			rule:        rule,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "add rule with startdatetime in past",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			rule:        ruleInPast,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "add rule with service error",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			rule:        rule,
			contentType: contentType,
			svcErr:      svcerr.ErrCreateEntity,
			status:      http.StatusUnprocessableEntity,
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.rule)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/rules", ts.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("AddRule", mock.Anything, tc.authnRes, tc.rule).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()

			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewRuleEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	cases := []struct {
		desc        string
		id          string
		domainID    string
		token       string
		contentType string
		status      int
		authnRes    smqauthn.Session
		authnErr    error
		svcRes      re.Rule
		svcErr      error
		err         error
		len         int
	}{
		{
			desc:        "view rule successfully",
			id:          rule.ID,
			token:       validToken,
			contentType: contentType,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			status:      http.StatusOK,
			svcRes:      rule,
		},
		{
			desc:        "view rule with invalid token",
			id:          rule.ID,
			token:       invalidToken,
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "view rule with empty token",
			token:       "",
			authnRes:    smqauthn.Session{},
			domainID:    domainID,
			id:          rule.ID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "view rule with empty domainID",
			token:       validToken,
			id:          rule.ID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "view rule with service error",
			token:       validToken,
			domainID:    domainID,
			authnRes:    smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID},
			id:          rule.ID,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/rules/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.authnRes, tc.authnErr)
			svcCall := svc.On("ViewRule", mock.Anything, tc.authnRes, tc.id).Return(tc.svcRes, tc.svcErr)
			res, err := req.make()

			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListRulesEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	cases := []struct {
		desc              string
		query             string
		domainID          string
		token             string
		session           smqauthn.Session
		listRulesResponse re.Page
		status            int
		authnErr          error
		err               error
	}{
		{
			desc:     "list rules successfully",
			domainID: domainID,
			token:    validToken,
			status:   http.StatusOK,
			listRulesResponse: re.Page{
				Total: 1,
				Rules: []re.Rule{rule},
			},
			err: nil,
		},
		{
			desc:     "list rules with empty token",
			domainID: domainID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list rules with invalid token",
			domainID: domainID,
			token:    invalidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list rules with offset",
			domainID: domainID,
			token:    validToken,
			listRulesResponse: re.Page{
				Total: 1,

				Rules: []re.Rule{rule},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list rules with invalid offset",
			domainID: domainID,
			token:    validToken,
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list rules with limit",
			domainID: domainID,
			token:    validToken,
			listRulesResponse: re.Page{
				Total: 1,

				Rules: []re.Rule{rule},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list rules with invalid limit",
			domainID: domainID,
			token:    validToken,
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list rules with invalid direction",
			domainID: domainID,
			token:    validToken,
			query:    "dir=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidDirection,
		},
		{
			desc:     "list rules with invalid order",
			domainID: domainID,
			token:    validToken,
			query:    "order=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidOrder,
		},
		{
			desc:     "list rule with limit that is too big",
			domainID: domainID,
			token:    validToken,
			query:    "limit=10000",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrLimitSize,
		},
		{
			desc:     "list rules with input channel",
			domainID: domainID,
			token:    validToken,
			listRulesResponse: re.Page{
				Total: 1,
				Rules: []re.Rule{rule},
			},
			query:  "input_channel=input.channel",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list rules with duplicate input_channel",
			domainID: domainID,
			token:    validToken,
			query:    "input_channel=1&input_channel=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list rules with status",
			domainID: domainID,
			token:    validToken,
			listRulesResponse: re.Page{
				Total: 1,
				Rules: []re.Rule{rule},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list rules with invalid status",
			domainID: domainID,
			token:    validToken,
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list rules with duplicate status",
			domainID: domainID,
			token:    validToken,
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list rules with duplicate tags",
			domainID: domainID,
			token:    validToken,
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:              "list rules with service error",
			domainID:          domainID,
			token:             validToken,
			listRulesResponse: re.Page{},
			status:            http.StatusForbidden,
			err:               svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodGet,
				url:         ts.URL + "/" + tc.domainID + "/rules?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListRules", mock.Anything, tc.session, mock.Anything).Return(tc.listRulesResponse, tc.err)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var bodyRes respBody
			err = json.NewDecoder(res.Body).Decode(&bodyRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if bodyRes.Err != "" || bodyRes.Message != "" {
				err = errors.Wrap(errors.New(bodyRes.Err), errors.New(bodyRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRulesEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	updateRuleReq := re.Rule{
		ID:   rule.ID,
		Name: rule.Name,
		Logic: re.Script{
			Type:  re.ScriptType(0),
			Value: "return `test` end",
		},
		InputChannel: testsutil.GenerateUUID(&testing.T{}),
		Metadata: map[string]any{
			"name": "test",
		},
	}

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		updateReq   re.Rule
		contentType string
		session     smqauthn.Session
		svcResp     re.Rule
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update rule successfully",
			token:       validToken,
			domainID:    domainID,
			id:          rule.ID,
			updateReq:   updateRuleReq,
			contentType: contentType,
			svcResp:     rule,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update rule with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          rule.ID,
			updateReq:   updateRuleReq,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update rule with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          rule.ID,
			updateReq:   updateRuleReq,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update rule with empty domainID",
			token:       validToken,
			id:          rule.ID,
			updateReq:   updateRuleReq,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "update rule with name that is too long",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			updateReq: re.Rule{
				ID:   validID,
				Name: strings.Repeat("a", 1025),
				Logic: re.Script{
					Type:  re.ScriptType(0),
					Value: "return `test` end",
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "update rule with invalid content type",
			token:       validToken,
			id:          rule.ID,
			domainID:    domainID,
			updateReq:   updateRuleReq,
			contentType: "application/xml",
			svcResp:     rule,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update rule with service error",
			token:       validToken,
			id:          rule.ID,
			domainID:    domainID,
			updateReq:   updateRuleReq,
			contentType: contentType,
			svcResp:     re.Rule{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.updateReq)
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/rules/%s", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateRule", mock.Anything, tc.session, tc.updateReq).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRuleTagsEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	newTag := "newtag"

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		contentType string
		session     smqauthn.Session
		svcResp     re.Rule
		svcErr      error
		resp        re.Rule
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update rule tags successfully",
			token:       validToken,
			domainID:    domainID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			svcResp:     rule,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update rule tags with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update rule tags with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update rule tags with empty domainID",
			token:       validToken,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "update rule tags with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			svcResp:     rule,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update rule tags with service error",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			svcResp:     re.Rule{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update rule with malformed request",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			contentType: contentType,
			data:        fmt.Sprintf(`{"tags":["%s"}`, newTag),
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "update rule with empty id",
			token:       validToken,
			id:          "",
			domainID:    domainID,
			contentType: contentType,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/rules/%s/tags", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateRuleTags", mock.Anything, tc.session, re.Rule{ID: tc.id, Tags: []string{newTag}}).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateRuleScheduleEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	updateScheduleReq := pkgSch.Schedule{
		StartDateTime:   future,
		Time:            future.Add(2 * time.Hour),
		Recurring:       pkgSch.Weekly,
		RecurringPeriod: 2,
	}

	ruleWithSchedule := rule
	ruleWithSchedule.Schedule = updateScheduleReq

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		schedule    pkgSch.Schedule
		contentType string
		session     smqauthn.Session
		svcResp     re.Rule
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update rule schedule successfully",
			token:       validToken,
			domainID:    domainID,
			id:          validID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			svcResp:     ruleWithSchedule,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update rule schedule with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update rule schedule with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    domainID,
			id:          validID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update rule schedule with empty domainID",
			token:       validToken,
			id:          validID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "update rule schedule with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			schedule:    updateScheduleReq,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:     "update rule schedule with start_datetime in past",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			schedule: pkgSch.Schedule{
				StartDateTime:   past,
				Time:            future,
				Recurring:       pkgSch.Daily,
				RecurringPeriod: 1,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "update rule schedule with service error",
			token:       validToken,
			id:          validID,
			domainID:    domainID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update rule schedule with empty id",
			token:       validToken,
			id:          "",
			domainID:    domainID,
			schedule:    updateScheduleReq,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(map[string]any{
				"schedule": tc.schedule,
			})

			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/rules/%s/schedule", ts.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}

			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateRuleSchedule", mock.Anything, mock.Anything, mock.Anything).Return(tc.svcResp, tc.svcErr)

			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableRuleEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  re.Rule
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "enable rule successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcResp:  rule,
			svcErr:   nil,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "enable rule with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "enable rule with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "enable rule with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "enable rule with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcResp:  re.Rule{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "enable rule with empty id",
			token:    validToken,
			id:       "",
			domainID: domainID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/rules/%s/enable", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableRule", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableRuleEndpoint(t *testing.T) {
	gs, svc, authn := newRuleEngineServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  re.Rule
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "disable rule successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcResp:  rule,
			svcErr:   nil,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "disable rule with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "disable rule with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "disable rule with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "disable rule with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcResp:  re.Rule{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "disable rule with empty id",
			token:    validToken,
			id:       "",
			domainID: domainID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/rules/%s/disable", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableRule", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			var errRes respBody
			err = json.NewDecoder(res.Body).Decode(&errRes)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error while decoding response body: %s", tc.desc, err))
			if errRes.Err != "" || errRes.Message != "" {
				err = errors.Wrap(errors.New(errRes.Err), errors.New(errRes.Message))
			}
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteRuleEndpoint(t *testing.T) {
	ts, svc, authn := newRuleEngineServer()
	defer ts.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "delete rule successfully",
			token:    validToken,
			domainID: domainID,
			id:       validID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "delete rule with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete rule with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: domainID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "delete rule with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "delete rule with service error",
			token:    validToken,
			id:       validID,
			domainID: domainID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: ts.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/rules/%s", ts.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: auth.EncodeDomainUserID(domainID, userID), UserID: userID, DomainID: domainID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveRule", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

type respBody struct {
	Err     string    `json:"error"`
	Message string    `json:"message"`
	Total   uint64    `json:"total"`
	ID      string    `json:"id"`
	Status  re.Status `json:"status"`
}
