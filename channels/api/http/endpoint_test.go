// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	"github.com/absmach/supermq/channels"
	"github.com/absmach/supermq/channels/mocks"
	"github.com/absmach/supermq/clients"
	"github.com/absmach/supermq/internal/testsutil"
	smqlog "github.com/absmach/supermq/logger"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/roles"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	valid            = "valid"
	validChannelResp = channels.Channel{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		Name:        valid,
		Domain:      testsutil.GenerateUUID(&testing.T{}),
		ParentGroup: testsutil.GenerateUUID(&testing.T{}),
		Metadata: clients.Metadata{
			"name": "test",
		},
		CreatedAt: time.Now().Add(-1 * time.Second),
		UpdatedAt: time.Now(),
		UpdatedBy: testsutil.GenerateUUID(&testing.T{}),
		Status:    clients.EnabledStatus,
	}
	validID      = testsutil.GenerateUUID(&testing.T{})
	validToken   = "validToken"
	invalidToken = "invalidToken"
	contentType  = "application/json"
)

func newChannelsServer() (*httptest.Server, *mocks.Service, *authnmocks.Authentication) {
	authn := new(authnmocks.Authentication)
	svc := new(mocks.Service)
	mux := chi.NewRouter()
	idp := uuid.NewMock()
	logger := smqlog.NewMock()
	mux = MakeHandler(svc, authn, mux, logger, "", idp)

	return httptest.NewServer(mux), svc, authn
}

func TestCreateChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	reqChannel := channels.Channel{
		Name: valid,
		Metadata: map[string]interface{}{
			"name": "test",
		},
	}

	cases := []struct {
		desc        string
		token       string
		session     smqauthn.Session
		domainID    string
		req         channels.Channel
		contentType string
		svcResp     []channels.Channel
		svcErr      error
		authnErr    error
		status      int
		err         error
	}{
		{
			desc:        "create channel successfully",
			token:       validToken,
			domainID:    validID,
			req:         reqChannel,
			contentType: contentType,
			svcResp:     []channels.Channel{validChannelResp},
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "create channel with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    validID,
			req:         reqChannel,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "create channel with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    validID,
			req:         reqChannel,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "create channel with empty domainID",
			token:       validToken,
			req:         reqChannel,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "create channel with name that is too long",
			token:    validToken,
			domainID: validID,
			req: channels.Channel{
				Name: strings.Repeat("a", 1025),
				Metadata: map[string]interface{}{
					"name": "test",
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "create channel with invalid content type",
			token:       validToken,
			domainID:    validID,
			req:         reqChannel,
			contentType: "application/xml",
			svcResp:     []channels.Channel{validChannelResp},
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "create channel with service error",
			token:       validToken,
			domainID:    validID,
			req:         reqChannel,
			contentType: contentType,
			svcResp:     []channels.Channel{},
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
				url:         fmt.Sprintf("%s/%s/channels/", gs.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("CreateChannels", mock.Anything, tc.session, tc.req).Return(tc.svcResp, []roles.RoleProvision{}, tc.svcErr)
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

func TestCreateChannelsEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	reqChannels := []channels.Channel{
		{
			Name: valid,
			Metadata: map[string]interface{}{
				"name": "test",
			},
		},
	}

	cases := []struct {
		desc        string
		token       string
		session     smqauthn.Session
		domainID    string
		req         []channels.Channel
		contentType string
		svcResp     []channels.Channel
		svcErr      error
		authnErr    error
		status      int
		err         error
	}{
		{
			desc:        "create channels successfully",
			token:       validToken,
			domainID:    validID,
			req:         reqChannels,
			contentType: contentType,
			svcResp:     []channels.Channel{validChannelResp},
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "create channels with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    validID,
			req:         reqChannels,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "create channels with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    validID,
			req:         reqChannels,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "create channels with empty domainID",
			token:       validToken,
			req:         reqChannels,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "create channels with name that is too long",
			token:    validToken,
			domainID: validID,
			req: []channels.Channel{
				{
					Name: strings.Repeat("a", 1025),
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
			desc:        "create channels with invalid content type",
			token:       validToken,
			domainID:    validID,
			req:         reqChannels,
			contentType: "application/xml",
			svcResp:     []channels.Channel{validChannelResp},
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "create channels with service error",
			token:       validToken,
			domainID:    validID,
			req:         reqChannels,
			contentType: contentType,
			svcResp:     []channels.Channel{},
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
				url:         fmt.Sprintf("%s/%s/channels/bulk", gs.URL, tc.domainID),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("CreateChannels", mock.Anything, tc.session, tc.req[0]).Return(tc.svcResp, []roles.RoleProvision{}, tc.svcErr)
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

func TestViewChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  channels.Channel
		svcErr   error
		resp     channels.Channel
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "view channel successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validChannelResp,
			svcErr:   nil,
			resp:     validChannelResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "view channel with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			svcResp:  validChannelResp,
			svcErr:   nil,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "view channel with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "view channel with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "view channel with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  validChannelResp,
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
				url:    fmt.Sprintf("%s/%s/channels/%s", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ViewChannel", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestListChannels(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc                 string
		query                string
		domainID             string
		token                string
		session              smqauthn.Session
		listChannelsResponse channels.Page
		status               int
		authnErr             error
		err                  error
	}{
		{
			desc:     "list channels successfully",
			domainID: validID,
			token:    validToken,
			status:   http.StatusOK,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			err: nil,
		},
		{
			desc:     "list channels with empty token",
			domainID: validID,
			token:    "",
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:     "list channels with invalid token",
			domainID: validID,
			token:    invalidToken,
			status:   http.StatusUnauthorized,
			authnErr: svcerr.ErrAuthentication,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "list channels with offset",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "offset=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid offset",
			domainID: validID,
			token:    validToken,
			query:    "offset=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with limit",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "limit=1",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid limit",
			domainID: validID,
			token:    validToken,
			query:    "limit=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with limit greater than max",
			token:    validToken,
			domainID: validID,
			query:    fmt.Sprintf("limit=%d", api.MaxLimitSize+1),
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with name",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "name=clientname",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid name",
			domainID: validID,
			token:    validToken,
			query:    "name=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate name",
			domainID: validID,
			token:    validToken,
			query:    "name=1&name=2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list channels with status",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "status=enabled",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid status",
			domainID: validID,
			token:    validToken,
			query:    "status=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate status",
			domainID: validID,
			token:    validToken,
			query:    "status=enabled&status=disabled",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list channels with tags",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "tag=tag1,tag2",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid tags",
			domainID: validID,
			token:    validToken,
			query:    "tag=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate tags",
			domainID: validID,
			token:    validToken,
			query:    "tag=tag1&tag=tag2",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list channels with metadata",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid metadata",
			domainID: validID,
			token:    validToken,
			query:    "metadata=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate metadata",
			domainID: validID,
			token:    validToken,
			query:    "metadata=%7B%22domain%22%3A%20%22example.com%22%7D&metadata=%7B%22domain%22%3A%20%22example.com%22%7D",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list channels with permissions",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "permission=view",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid permissions",
			domainID: validID,
			token:    validToken,
			query:    "permission=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate permissions",
			domainID: validID,
			token:    validToken,
			query:    "permission=view&permission=view",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrInvalidQueryParams,
		},
		{
			desc:     "list channels with list perms",
			domainID: validID,
			token:    validToken,
			listChannelsResponse: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{validChannelResp},
			},
			query:  "list_perms=true",
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:     "list channels with invalid list perms",
			domainID: validID,
			token:    validToken,
			query:    "list_perms=invalid",
			status:   http.StatusBadRequest,
			err:      apiutil.ErrValidation,
		},
		{
			desc:     "list channels with duplicate list perms",
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
				url:         gs.URL + "/" + tc.domainID + "/channels?" + tc.query,
				contentType: contentType,
				token:       tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("ListChannels", mock.Anything, tc.session, mock.Anything).Return(tc.listChannelsResponse, tc.err)
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

func TestUpdateChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	updateChannelReq := channels.Channel{
		ID:   validID,
		Name: valid,
		Metadata: map[string]interface{}{
			"name": "test",
		},
	}

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		updateReq   channels.Channel
		contentType string
		session     smqauthn.Session
		svcResp     channels.Channel
		svcErr      error
		resp        channels.Channel
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update channel successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			updateReq:   updateChannelReq,
			contentType: contentType,
			svcResp:     validChannelResp,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update channel with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    validID,
			id:          validID,
			updateReq:   updateChannelReq,
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update channel with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    validID,
			id:          validID,
			updateReq:   updateChannelReq,
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update channel with empty domainID",
			token:       validToken,
			id:          validID,
			updateReq:   updateChannelReq,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:     "update channel with name that is too long",
			token:    validToken,
			id:       validID,
			domainID: validID,
			updateReq: channels.Channel{
				ID:   validID,
				Name: strings.Repeat("a", 1025),
				Metadata: map[string]interface{}{
					"name": "test",
				},
			},
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrNameSize,
		},
		{
			desc:        "update channel with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			updateReq:   updateChannelReq,
			contentType: "application/xml",
			svcResp:     validChannelResp,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update channel with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			updateReq:   updateChannelReq,
			contentType: contentType,
			svcResp:     channels.Channel{},
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
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/channels/%s", gs.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateChannel", mock.Anything, tc.session, tc.updateReq).Return(tc.svcResp, tc.svcErr)
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

func TestUpdateChannelTagsEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	newTag := "newtag"

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		contentType string
		session     smqauthn.Session
		svcResp     channels.Channel
		svcErr      error
		resp        channels.Channel
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "update channel tags successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			svcResp:     validChannelResp,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "update channel tags with invalid token",
			token:       invalidToken,
			session:     smqauthn.Session{},
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "update channel tags with empty token",
			token:       "",
			session:     smqauthn.Session{},
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "update channel tags with empty domainID",
			token:       validToken,
			id:          validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "update channel tags with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: "application/xml",
			svcResp:     validChannelResp,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "update channel tags with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			contentType: contentType,
			svcResp:     channels.Channel{},
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "update channel with malformed request",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			contentType: contentType,
			data:        fmt.Sprintf(`{"tags":["%s"}`, newTag),
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "update channel with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			contentType: contentType,
			data:        fmt.Sprintf(`{"tags":["%s"]}`, newTag),
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPatch,
				url:         fmt.Sprintf("%s/%s/channels/%s/tags", gs.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("UpdateChannelTags", mock.Anything, tc.session, channels.Channel{ID: tc.id, Tags: []string{newTag}}).Return(tc.svcResp, tc.svcErr)
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

func TestSetChannelParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		contentType string
		session     smqauthn.Session
		svcErr      error
		resp        channels.Channel
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "set channel parent group successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusOK,
			err:         nil,
		},
		{
			desc:        "set channel parent group with invalid token",
			token:       invalidToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			authnErr:    svcerr.ErrAuthentication,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:        "set channel parent group with empty token",
			token:       "",
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			err:         apiutil.ErrBearerToken,
		},
		{
			desc:        "set channel parent group with empty domainID",
			token:       validToken,
			id:          validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingDomainID,
		},
		{
			desc:        "set channel parent group with invalid content type",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: "application/xml",
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
		},
		{
			desc:        "set channel parent group with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
		{
			desc:        "set channel parent group with empty parent group id",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        `{"parent_group_id":""}`,
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingParentGroupID,
		},
		{
			desc:        "set channel parent group with malformed request",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         errors.ErrMalformedEntity,
		},
		{
			desc:        "set channel parent group with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"parent_group_id":"%s"}`, validID),
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/parent", gs.URL, tc.domainID, tc.id),
				contentType: tc.contentType,
				token:       tc.token,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("SetParentGroup", mock.Anything, tc.session, validID, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveChannelParentGroupEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcErr   error
		resp     channels.Channel
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "remove channel parent group successfully",
			token:    validToken,
			id:       validID,
			domainID: validID,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "remove channel parent group with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			id:       validID,
			domainID: validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:   "remove channel parent group with empty token",
			token:  "",
			id:     validID,
			status: http.StatusUnauthorized,
			err:    apiutil.ErrBearerToken,
		},
		{
			desc:   "remove channel parent group with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "remove channel parent group with empty id",
			token:    validToken,
			id:       "",
			domainID: validID,
			status:   http.StatusBadRequest,
			err:      apiutil.ErrMissingID,
		},
		{
			desc:     "remove channel parent group with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client: gs.Client(),
				method: http.MethodDelete,
				url:    fmt.Sprintf("%s/%s/channels/%s/parent", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
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

func TestEnableChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  channels.Channel
		svcErr   error
		resp     channels.Channel
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "enable channel successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validChannelResp,
			svcErr:   nil,
			resp:     validChannelResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "enable channel with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "enable channel with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "enable channel with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "enable channel with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  channels.Channel{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "enable channel with empty id",
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
				url:    fmt.Sprintf("%s/%s/channels/%s/enable", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("EnableChannel", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestDisableChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc     string
		token    string
		id       string
		domainID string
		session  smqauthn.Session
		svcResp  channels.Channel
		svcErr   error
		resp     channels.Channel
		status   int
		authnErr error
		err      error
	}{
		{
			desc:     "disable channel successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcResp:  validChannelResp,
			svcErr:   nil,
			resp:     validChannelResp,
			status:   http.StatusOK,
			err:      nil,
		},
		{
			desc:     "disable channel with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "disable channel with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "disable channel with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "disable channel with service error",
			token:    validToken,
			id:       validID,
			domainID: validID,
			svcResp:  channels.Channel{},
			svcErr:   svcerr.ErrAuthorization,
			status:   http.StatusForbidden,
			err:      svcerr.ErrAuthorization,
		},
		{
			desc:     "disable channel with empty id",
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
				url:    fmt.Sprintf("%s/%s/channels/%s/disable", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("DisableChannel", mock.Anything, tc.session, tc.id).Return(tc.svcResp, tc.svcErr)
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

func TestConnectChannelClientEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		session     smqauthn.Session
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "connect channel client successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			svcErr:      nil,
			status:      http.StatusCreated,
			err:         nil,
		},
		{
			desc:        "connect channel client with invalid token",
			token:       invalidToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "connect channel client with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "connect channel client with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:        "connect channel client with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "connect channel client with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/connect", gs.URL, tc.domainID, tc.id),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("Connect", mock.Anything, tc.session, []string{tc.id}, []string{validID}, []connections.ConnType{1}).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnectChannelClientEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc        string
		token       string
		id          string
		domainID    string
		data        string
		session     smqauthn.Session
		contentType string
		svcErr      error
		status      int
		authnErr    error
		err         error
	}{
		{
			desc:        "disconnect channel client successfully",
			token:       validToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			svcErr:      nil,
			status:      http.StatusNoContent,
			err:         nil,
		},
		{
			desc:        "disconnect channel client with invalid token",
			token:       invalidToken,
			domainID:    validID,
			id:          validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			status:      http.StatusUnauthorized,
			authnErr:    svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:     "disconnect channel client with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "disconnect channel client with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:        "disconnect channel client with service error",
			token:       validToken,
			id:          validID,
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			svcErr:      svcerr.ErrAuthorization,
			status:      http.StatusForbidden,
			err:         svcerr.ErrAuthorization,
		},
		{
			desc:        "disconnect channel client with empty id",
			token:       validToken,
			id:          "",
			domainID:    validID,
			data:        fmt.Sprintf(`{"client_ids": ["%s"], "types": ["Publish"]}`, validID),
			contentType: contentType,
			status:      http.StatusBadRequest,
			err:         apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/%s/disconnect", gs.URL, tc.domainID, tc.id),
				token:       tc.token,
				contentType: tc.contentType,
				body:        strings.NewReader(tc.data),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("Disconnect", mock.Anything, tc.session, []string{tc.id}, []string{validID}, []connections.ConnType{1}).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestConnectEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc       string
		token      string
		channelIDs []string
		domainID   string
		clientIDs  []string
		types      []connections.ConnType
		session    smqauthn.Session
		svcErr     error
		status     int
		authnErr   error
		err        error
	}{
		{
			desc:       "connect successfully",
			token:      validToken,
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			svcErr:     nil,
			status:     http.StatusCreated,
			err:        nil,
		},
		{
			desc:       "connect with invalid token",
			token:      invalidToken,
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusUnauthorized,
			authnErr:   svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "connect with empty token",
			token:      "",
			session:    smqauthn.Session{},
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusUnauthorized,
			err:        apiutil.ErrBearerToken,
		},
		{
			desc:       "connect with empty domainID",
			token:      validToken,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingDomainID,
		},
		{
			desc:       "connect with service error",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			svcErr:     svcerr.ErrAuthorization,
			status:     http.StatusForbidden,
			err:        svcerr.ErrAuthorization,
		},
		{
			desc:       "connect with empty channel ids",
			token:      validToken,
			channelIDs: []string{},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
		{
			desc:       "connect with empty client ids",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
		{
			desc:       "connect with empty types",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/connect", gs.URL, tc.domainID),
				token:       tc.token,
				contentType: contentType,
				body: strings.NewReader(toJSON(map[string]interface{}{
					"channel_ids": tc.channelIDs,
					"client_ids":  tc.clientIDs,
					"types":       tc.types,
				})),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("Connect", mock.Anything, tc.session, tc.channelIDs, tc.clientIDs, tc.types).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnectEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

	cases := []struct {
		desc       string
		token      string
		channelIDs []string
		domainID   string
		clientIDs  []string
		types      []connections.ConnType
		session    smqauthn.Session
		svcErr     error
		status     int
		authnErr   error
		err        error
	}{
		{
			desc:       "disconnect successfully",
			token:      validToken,
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			svcErr:     nil,
			status:     http.StatusNoContent,
			err:        nil,
		},
		{
			desc:       "disconnect with invalid token",
			token:      invalidToken,
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusUnauthorized,
			authnErr:   svcerr.ErrAuthentication,
			err:        svcerr.ErrAuthentication,
		},
		{
			desc:       "disconnect with empty token",
			token:      "",
			session:    smqauthn.Session{},
			domainID:   validID,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusUnauthorized,
			err:        apiutil.ErrBearerToken,
		},
		{
			desc:       "disconnect with empty domainID",
			token:      validToken,
			channelIDs: []string{validID},
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingDomainID,
		},
		{
			desc:       "disconnect with service error",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			svcErr:     svcerr.ErrAuthorization,
			status:     http.StatusForbidden,
			err:        svcerr.ErrAuthorization,
		},
		{
			desc:       "disconnect with empty channel ids",
			token:      validToken,
			channelIDs: []string{},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
		{
			desc:       "disconnect with empty client ids",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{},
			types:      []connections.ConnType{1},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
		{
			desc:       "disconnect with empty types",
			token:      validToken,
			channelIDs: []string{validID},
			domainID:   validID,
			clientIDs:  []string{validID},
			types:      []connections.ConnType{},
			status:     http.StatusBadRequest,
			err:        apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			req := testRequest{
				client:      gs.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/%s/channels/disconnect", gs.URL, tc.domainID),
				token:       tc.token,
				contentType: contentType,
				body: strings.NewReader(toJSON(map[string]interface{}{
					"channel_ids": tc.channelIDs,
					"client_ids":  tc.clientIDs,
					"types":       tc.types,
				})),
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("Disconnect", mock.Anything, tc.session, tc.channelIDs, tc.clientIDs, tc.types).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteChannelEndpoint(t *testing.T) {
	gs, svc, authn := newChannelsServer()
	defer gs.Close()

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
			desc:     "delete channel successfully",
			token:    validToken,
			domainID: validID,
			id:       validID,
			svcErr:   nil,
			status:   http.StatusNoContent,
			err:      nil,
		},
		{
			desc:     "delete channel with invalid token",
			token:    invalidToken,
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			authnErr: svcerr.ErrAuthentication,
			status:   http.StatusUnauthorized,
			err:      svcerr.ErrAuthentication,
		},
		{
			desc:     "delete channel with empty token",
			token:    "",
			session:  smqauthn.Session{},
			domainID: validID,
			id:       validID,
			status:   http.StatusUnauthorized,
			err:      apiutil.ErrBearerToken,
		},
		{
			desc:   "delete channel with empty domainID",
			token:  validToken,
			id:     validID,
			status: http.StatusBadRequest,
			err:    apiutil.ErrMissingDomainID,
		},
		{
			desc:     "delete channel with service error",
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
				url:    fmt.Sprintf("%s/%s/channels/%s", gs.URL, tc.domainID, tc.id),
				token:  tc.token,
			}
			if tc.token == validToken {
				tc.session = smqauthn.Session{DomainUserID: validID + "_" + validID, UserID: validID, DomainID: validID}
			}
			authCall := authn.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authnErr)
			svcCall := svc.On("RemoveChannel", mock.Anything, tc.session, tc.id).Return(tc.svcErr)
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
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
	Err         string         `json:"error"`
	Message     string         `json:"message"`
	Total       int            `json:"total"`
	Permissions []string       `json:"permissions"`
	ID          string         `json:"id"`
	Tags        []string       `json:"tags"`
	Status      clients.Status `json:"status"`
}
