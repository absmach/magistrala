// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/groups/mocks"
	mgapi "github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	validGroupResp = groups.Group{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        valid,
		Description: valid,
		Domain:      testsutil.GenerateUUID(&testing.T{}),
		Parent:      testsutil.GenerateUUID(&testing.T{}),
		Metadata: groups.Metadata{
			"name": "test",
		},
		Children:  []*groups.Group{},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(&testing.T{}),
		Status:    groups.EnabledStatus,
	}
	validID      = testsutil.GenerateUUID(&testing.T{})
	validToken   = "validToken"
	invalidToken = "invalidToken"
	contentType  = "application/json"
)

func newGroupsServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	authn := new(authnmocks.Authentication)
	svc := new(mocks.Service)
	mux := chi.NewRouter()
	logger := mglog.NewMock()
	mux = MakeHandler(svc, authn, mux, logger, "")

	return httptest.NewServer(mux), svc, authn
}

func TestCreateGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	reqGroup := groups.Group{
		Name:        valid,
		Description: valid,
		Metadata: map[string]interface{}{
			"name": "test",
		},
	}

	cases := []struct {
		desc        string
		token       string
		session     mgauthn.Session
		domainID    string
		req         createGroupReq
		contentType string
		svcResp     groups.Group
		svcErr      error
		authnErr    error
		status      int
		err         error
	}{
		{
			desc:     "create group successfully",
			token:    validToken,
			domainID: validID,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: contentType,
			svcResp:     validGroupResp,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:     "create group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "create group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:  "create group with empty domainID",
			token: validToken,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "create group with missing name",
			token:    validToken,
			domainID: validID,
			req: createGroupReq{
				Group: groups.Group{
					Description: valid,
					Metadata: map[string]interface{}{
						"name": "test",
					},
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:     "create group with name that is too long",
			token:    validToken,
			domainID: validID,
			req: createGroupReq{
				Group: groups.Group{
					Name:        strings.Repeat("a", 1025),
					Description: valid,
					Metadata: map[string]interface{}{
						"name": "test",
					},
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:     "create group with invalid content type",
			token:    validToken,
			domainID: validID,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: "application/xml",
			svcResp:     validGroupResp,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:     "create group with service error",
			token:    validToken,
			domainID: validID,
			req: createGroupReq{
				Group: reqGroup,
			},
			contentType: contentType,
			svcResp:     groups.Group{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.req)
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/groups/", gs.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("CreateGroup", mock.Anything, tc.session, tc.req.Group).Return(tc.svcResp, tc.svcErr)
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

func TestViewGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcResp  groups.Group
		svcErr   error
		resp     groups.Group
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "view group successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validGroupResp,
			svcErr:   nil,
			resp:     validGroupResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			svcResp:  validGroupResp,
			svcErr:   nil,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "view group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "view group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  validGroupResp,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/groups/%s", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ViewGroup", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestUpdateGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	updateGroupReq := groups.Group{
		ID:          validID,
		Name:        valid,
		Description: valid,
		Metadata: map[string]interface{}{
			"name": "test",
		},
	}

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		updateReq   groups.Group
		contentType string
		session     mgauthn.Session
		svcResp     groups.Group
		svcErr      error
		resp        groups.Group
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update group successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			updateReq:   updateGroupReq,
			contentType: contentType,
			svcResp:     validGroupResp,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update group with invalid token",
			token:       invalidToken,
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validID,
			updateReq:   updateGroupReq,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update group with empty token",
			token:       "",
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validID,
			updateReq:   updateGroupReq,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update group with empty domainID",
			token:       validToken,
			id:          validID,
			updateReq:   updateGroupReq,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "update group with name that is too long",
			token:    validToken,
			id:       validID,
			domainID: validID,
			updateReq: groups.Group{
				ID:          validID,
				Name:        strings.Repeat("a", 1025),
				Description: valid,
				Metadata: map[string]interface{}{
					"name": "test",
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "update group with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			updateReq:   updateGroupReq,
			contentType: "application/xml",
			svcResp:     validGroupResp,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update group with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			updateReq:   updateGroupReq,
			contentType: contentType,
			svcResp:     groups.Group{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			data := toJSON(tc.updateReq)
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPut,
				url:         fmt.Sprintf("%s/%s/groups/%s", gs.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateGroup", mock.Anything, tc.session, tc.updateReq).Return(tc.svcResp, tc.svcErr)
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

func TestEnableGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcResp  groups.Group
		svcErr   error
		resp     groups.Group
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "enable group successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validGroupResp,
			svcErr:   nil,
			resp:     validGroupResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "enable group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "enable group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "enable group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "enable group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  groups.Group{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "enable group with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/groups/%s/enable", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableGroup", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestDisableGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcResp  groups.Group
		svcErr   error
		resp     groups.Group
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "disable group successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validGroupResp,
			svcErr:   nil,
			resp:     validGroupResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "disable group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "disable group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "disable group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "disable group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  groups.Group{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "disable group with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodPost,
				url:    fmt.Sprintf("%s/%s/groups/%s/disable", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableGroup", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestListGroups(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc               string
		query              string
		domainID           string
		token              string
		session            mgauthn.Session
		listGroupsResponse groups.Page
		status             int
		authnErr           error
		err                error
	}{
		{
			desc:     "list groups successfully",
			domainID: validID,
			token:    validToken,
			status:   http.StatusOK,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			err: nil,
		},
		{
			desc:     "list groups with empty token",
			domainID: validID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list groups with invalid token",
			domainID: validID,
			token:    invalidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list groups with offset",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid offset",
			domainID: validID,
			token:    validToken,
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with limit",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid limit",
			domainID: validID,
			token:    validToken,
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with limit greater than max",
			token:    validToken,
			domainID: validID,
			query:    fmt.Sprintf("limit=%d", mgapi.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with name",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "name=clientname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid name",
			domainID: validID,
			token:    validToken,
			query:    "name=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate name",
			domainID: validID,
			token:    validToken,
			query:    "name=1&name=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list groups with status",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid status",
			domainID: validID,
			token:    validToken,
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate status",
			domainID: validID,
			token:    validToken,
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list groups with tags",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid tags",
			domainID: validID,
			token:    validToken,
			query:    "tag=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate tags",
			domainID: validID,
			token:    validToken,
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list groups with metadata",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid metadata",
			domainID: validID,
			token:    validToken,
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate metadata",
			domainID: validID,
			token:    validToken,
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list groups with permissions",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid permissions",
			domainID: validID,
			token:    validToken,
			query:    "permission=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate permissions",
			domainID: validID,
			token:    validToken,
			query:    "permission=view&permission=view",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list groups with list perms",
			domainID: validID,
			token:    validToken,
			listGroupsResponse: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			query:  "list_perms=true",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list groups with invalid list perms",
			domainID: validID,
			token:    validToken,
			query:    "list_perms=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list groups with duplicate list perms",
			domainID: validID,
			token:    validToken,
			query:    "list_perms=true&listPerms=true",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodGet,
				url:         gs.URL + "/" + tc.domainID + "/groups?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListGroups", mock.Anything, tc.session, mock.Anything).Return(tc.listGroupsResponse, tc.err)
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

func TestDeleteGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "delete group successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "delete group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "delete group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "delete group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/groups/%s", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DeleteGroup", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRetrieveGroupHierarchyEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	retrieveHierarchRes := groups.HierarchyPage{
		Groups: []groups.Group{validGroupResp},
		HierarchyPageMeta: groups.HierarchyPageMeta{
			Level:     1,
			Direction: -1,
			Tree:      false,
		},
	}

	cases := []struct {
		desc     string
		token    string
		session  mgauthn.Session
		domainID string
		groupID  string
		query    string
		pageMeta groups.HierarchyPageMeta
		svcRes   groups.HierarchyPage
		svcErr   error
		authnErr error
		status   int
		err      error
	}{
		{
			desc:     "retrieve group hierarchy successfully",
			token:    validToken,
			domainID: validID,
			groupID:  validID,
			query:    "level=1&dir=-1&tree=false",
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			svcRes: retrieveHierarchRes,
			svcErr: nil,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "retrieve group hierarchy with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			groupID:  validID,
			query:    "level=1&dir=-1&tree=false",
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:    "retrieve group hierarchy with empty token",
			token:   "",
			session: mgauthn.Session{},
			status:  http.StatusUnauthorized,
			err:     apiutil.ErrBearerToken,
		},
		{
			desc:    "retrieve group hierarchy with empty domainID",
			token:   validToken,
			groupID: validID,
			query:   "level=1&dir=-1&tree=false",
			status:  http.StatusBadRequest,
			err:     apiutil.ErrMissingDomainID,
		},
		{
			desc:     "retrieve group hierarchy with service error",
			token:    validToken,
			groupID:  validID,
			domainID: validID,
			query:    "level=1&dir=-1&tree=false",
			pageMeta: groups.HierarchyPageMeta{
				Level:     1,
				Direction: -1,
				Tree:      false,
			},
			svcRes: groups.HierarchyPage{},
			svcErr: svcerr.ErrAuthorization,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc:     "retrieve group hierarchy with invalid level",
			token:    validToken,
			groupID:  validID,
			domainID: validID,
			query:    "level=invalid&dir=-1&tree=false",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "retrieve group hierarchy with invalid direction",
			token:    validToken,
			groupID:  validID,
			domainID: validID,
			query:    "level=1&dir=invalid&tree=false",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "retrieve group hierarchy with invalid tree",
			token:    validToken,
			groupID:  validID,
			domainID: validID,
			query:    "level=1&dir=-1&tree=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "retrieve group hierarchy with empty groupID",
			token:    validToken,
			domainID: validID,
			query:    "level=1&dir=-1&tree=false",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/groups/%s/hierarchy?%s", gs.URL, tc.domainID, tc.groupID, tc.query),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RetrieveGroupHierarchy", mock.Anything, tc.session, tc.groupID, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
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

func TestAddParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		parentID    string
		session     mgauthn.Session
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "add parent group successfully",
			token:       validToken,
			domainID:    validID,
			id:          validGroupResp.ID,
			parentID:    validID,
			contentType: contentType,
			svcErr:      nil,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "add parent group with invalid token",
			token:       invalidToken,
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			parentID:    validID,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "add parent group with empty token",
			token:       "",
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			parentID:    validID,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "add parent group with empty domainID",
			token:       validToken,
			id:          validGroupResp.ID,
			parentID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "add parent group with service error",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			parentID:    validID,
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "add parent group with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			parentID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "add parent group with empty parentID",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "add self parenting group",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			parentID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrSelfParentingNotAllowed,
		},
		{
			desc:        "add parent group with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			parentID:    validID,
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			reqData := struct {
				ParentID string `json:"parent_id"`
			}{
				ParentID: tc.parentID,
			}
			data := toJSON(reqData)
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/groups/%s/parent", gs.URL, tc.domainID, tc.id),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("AddParentGroup", mock.Anything, tc.session, tc.id, tc.parentID).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "remove parent group successfully",
			token:    validToken,
			domainID: validID,
			id:       validGroupResp.ID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "remove parent group with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "remove parent group with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "remove parent group with empty domainID",
			token:  validToken,
			id:     validGroupResp.ID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "remove parent group with service error",
			token:    validToken,
			id:       validGroupResp.ID,
			domainID: validID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "remove parent group with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/groups/%s/parent", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveParentGroup", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddChildrenGroupsEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		childrenIDs []string
		session     mgauthn.Session
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "add children groups successfully",
			token:       validToken,
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			svcErr:      nil,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "add children groups with invalid token",
			token:       invalidToken,
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "add children groups with empty token",
			token:       "",
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "add children groups with empty domainID",
			token:       validToken,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "add children groups with service error",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "add children groups with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "add children groups with empty childrenIDs",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "add children groups with invalid childrenIDs",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{"invalid"},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "add self children group",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrSelfParentingNotAllowed,
		},
		{
			desc:        "add children groups with invalid content type",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			reqData := struct {
				ChildrenIDs []string `json:"children_ids"`
			}{
				ChildrenIDs: tc.childrenIDs,
			}
			data := toJSON(reqData)
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/groups/%s/children", gs.URL, tc.domainID, tc.id),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("AddChildrenGroups", mock.Anything, tc.session, tc.id, tc.childrenIDs).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveChildrenGroupsEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		session     mgauthn.Session
		childrenIDs []string
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "remove children groups successfully",
			token:       validToken,
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			svcErr:      nil,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "remove children groups with invalid token",
			token:       invalidToken,
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "remove children groups with empty token",
			token:       "",
			session:     mgauthn.Session{},
			domainID:    validID,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "remove children groups with empty domainID",
			token:       validToken,
			id:          validGroupResp.ID,
			childrenIDs: []string{validID},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "remove children groups with service error",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "remove children groups with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			contentType: contentType,
			childrenIDs: []string{validID},
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "remove children groups with empty childrenIDs",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "remove children groups with invalid childrenIDs",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{"invalid"},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrValidation,
		},
		{
			desc:        "remove children groups with invalid content type",
			token:       validToken,
			id:          validGroupResp.ID,
			domainID:    validID,
			childrenIDs: []string{validID},
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			reqData := struct {
				ChildrenIDs []string `json:"children_ids"`
			}{
				ChildrenIDs: tc.childrenIDs,
			}
			data := toJSON(reqData)
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodDelete,
				url:         fmt.Sprintf("%s/%s/groups/%s/children", gs.URL, tc.domainID, tc.id),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveChildrenGroups", mock.Anything, tc.session, tc.id, tc.childrenIDs).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveAllChildrenGroupsEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		svcErr   error
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "remove all children groups successfully",
			token:    validToken,
			domainID: validID,
			id:       validGroupResp.ID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "remove all children groups with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "remove all children groups with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "remove all children groups with empty domainID",
			token:  validToken,
			id:     validGroupResp.ID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "remove all children groups with service error",
			token:    validToken,
			id:       validGroupResp.ID,
			domainID: validID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "remove all children groups with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/groups/%s/children/all", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveAllChildrenGroups", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListChildrenGroupsEndpoint(t *testing.T) {
	gs, svc, authn := newGroupsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  mgauthn.Session
		query    string
		pageMeta groups.PageMeta
		svcRes   groups.Page
		svcErr   error
		authnErr error
		status   int
		err      error
	}{
		{
			desc:     "list children groups successfully",
			token:    validToken,
			domainID: validID,
			id:       validGroupResp.ID,
			query:    "limit=1&offset=0",
			pageMeta: groups.PageMeta{
				Limit:   1,
				Offset:  0,
				Actions: []string{},
			},
			svcRes: groups.Page{
				PageMeta: groups.PageMeta{
					Total: 1,
				},
				Groups: []groups.Group{validGroupResp},
			},
			svcErr: nil,
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list children groups with invalid token",
			token:    invalidToken,
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			query:    "limit=1&offset=0",
			pageMeta: groups.PageMeta{
				Limit:   1,
				Offset:  0,
				Actions: []string{},
			},
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list children groups with empty token",
			token:    "",
			session:  mgauthn.Session{},
			domainID: validID,
			id:       validGroupResp.ID,
			query:    "limit=1&offset=0",
			pageMeta: groups.PageMeta{
				Limit:   1,
				Offset:  0,
				Actions: []string{},
			},
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "list children groups with empty domainID",
			token:  validToken,
			id:     validGroupResp.ID,
			query:  "limit=1&offset=0",
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "list children groups with service error",
			token:    validToken,
			id:       validGroupResp.ID,
			domainID: validID,
			query:    "limit=1&offset=0",
			pageMeta: groups.PageMeta{
				Limit:   1,
				Offset:  0,
				Actions: []string{},
			},
			svcRes: groups.Page{},
			svcErr: svcerr.ErrAuthorization,
			status: http.StatusForbidden,
			err:    svcerr.ErrAuthorization,
		},
		{
			desc:     "list children groups with invalid limit",
			token:    validToken,
			id:       validGroupResp.ID,
			domainID: validID,
			query:    "limit=invalid&offset=0",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list children groups with invalid offset",
			token:    validToken,
			id:       validGroupResp.ID,
			domainID: validID,
			query:    "limit=1&offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list children groups with empty id",
			token:    validToken,
			domainID: validID,
			query:    "limit=1&offset=0",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodGet,
				url:    fmt.Sprintf("%s/%s/groups/%s/children?%s", gs.URL, tc.domainID, tc.id, tc.query),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListChildrenGroups", mock.Anything, tc.session, tc.id, int64(1), int64(0), tc.pageMeta).Return(tc.svcRes, tc.svcErr)
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

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

type respBody struct {
	Err         string        `json:"error"`
	Message     string        `json:"message"`
	Total       int           `json:"total"`
	Permissions []string      `json:"permissions"`
	ID          string        `json:"id"`
	Tags        []string      `json:"tags"`
	Status      groups.Status `json:"status"`
}
