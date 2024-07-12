// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/twins"
	httpapi "github.com/absmach/magistrala/twins/api/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	twinName     = "name"
	contentType  = "application/json"
	email        = "user@example.com"
	token        = "token"
	invalidtoken = "invalid"
	wrongID      = 0
	maxNameSize  = 1024
	instanceID   = "5de9b29a-feb9-11ed-be56-0242ac120002"
	retained     = "saved"
	validID      = "123e4567-e89b-12d3-a456-426614174000"
)

var invalidName = strings.Repeat("m", maxNameSize+1)

type twinReq struct {
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type twinRes struct {
	Owner    string                 `json:"owner"`
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Revision int                    `json:"revision"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type twinsPageRes struct {
	pageRes
	Twins []twinRes `json:"twins"`
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
	return tr.client.Do(req)
}

func newServer(svc twins.Service) *httptest.Server {
	logger := mglog.NewMock()
	mux := httpapi.MakeHandler(svc, logger, instanceID)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func TestAddTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()
	ts := newServer(svc)
	defer ts.Close()

	tw := twinReq{}
	data, err := toJSON(tw)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	tw.Name = invalidName
	invalidData, err := toJSON(tw)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
		err         error
		saveErr     error
		identifyErr error
		userID      string
	}{
		{
			desc:        "add valid twin",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/twins/123e4567-e89b-12d3-a456-000000000001",
			err:         nil,
			saveErr:     nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/twins/123e4567-e89b-12d3-a456-000000000002",
			err:         nil,
			saveErr:     nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin with invalid auth token",
			req:         data,
			contentType: contentType,
			auth:        invalidtoken,
			status:      http.StatusUnauthorized,
			location:    "",
			err:         svcerr.ErrAuthentication,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "add twin with empty auth token",
			req:         data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			location:    "",
			err:         svcerr.ErrAuthentication,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "add twin with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrMalformedEntity,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrMalformedEntity,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
			err:         apiutil.ErrUnsupportedContentType,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "add twin with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
			err:         svcerr.ErrMalformedEntity,
			saveErr:     svcerr.ErrCreateEntity,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("Save", mock.Anything, mock.Anything).Return(retained, tc.saveErr)
		cacheCall := twinCache.On("Save", mock.Anything, mock.Anything).Return(tc.err)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/twins", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))
		authCall.Unset()
		repoCall.Unset()
		cacheCall.Unset()
	}
}

func TestUpdateTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()
	ts := newServer(svc)
	defer ts.Close()

	twin := twins.Twin{
		Owner: email,
		ID:    testsutil.GenerateUUID(t),
	}
	twin.Name = twinName
	data, err := toJSON(twin)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	tw := twin
	tw.Name = invalidName
	invalidData, err := toJSON(tw)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
		err         error
		retrieveErr error
		updateErr   error
		identifyErr error
		userID      string
	}{
		{
			desc:        "update existing twin",
			req:         data,
			id:          twin.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
			err:         nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update twin with empty JSON request",
			req:         "{}",
			id:          twin.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
			retrieveErr: nil,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update non-existent twin",
			req:         data,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
			err:         svcerr.ErrNotFound,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update twin with invalid token",
			req:         data,
			id:          twin.ID,
			contentType: contentType,
			auth:        invalidtoken,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "update twin with empty token",
			req:         data,
			id:          twin.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "update twin with invalid data format",
			req:         "{",
			id:          twin.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			retrieveErr: nil,
			updateErr:   svcerr.ErrUpdateEntity,
			userID:      validID,
		},
		{
			desc:        "update twin with empty request",
			req:         "",
			id:          twin.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			err:         svcerr.ErrMalformedEntity,
			retrieveErr: nil,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update twin without content type",
			req:         data,
			id:          twin.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			err:         apiutil.ErrUnsupportedContentType,
			retrieveErr: nil,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "update twin with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusMethodNotAllowed,
			err:         svcerr.ErrMalformedEntity,
			retrieveErr: svcerr.ErrNotFound,
			updateErr:   svcerr.ErrUpdateEntity,
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("RetrieveByID", mock.Anything, tc.id).Return(twins.Twin{}, tc.retrieveErr)
		repoCall1 := twinRepo.On("Update", mock.Anything, mock.Anything).Return(tc.updateErr)
		cacheCall := twinCache.On("Update", mock.Anything, mock.Anything).Return(tc.err)
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		authCall.Unset()
		repoCall.Unset()
		repoCall1.Unset()
		cacheCall.Unset()
	}
}

func TestViewTwin(t *testing.T) {
	svc, auth, twinRepo, _, _ := NewService()
	ts := newServer(svc)
	defer ts.Close()

	twin := twins.Twin{
		Owner:    email,
		ID:       testsutil.GenerateUUID(t),
		Name:     twinName,
		Revision: 50,
	}

	twres := twinRes{
		Owner:    twin.Owner,
		Name:     twin.Name,
		ID:       twin.ID,
		Revision: twin.Revision,
		Metadata: twin.Metadata,
	}

	cases := []struct {
		desc        string
		id          string
		auth        string
		status      int
		res         twinRes
		err         error
		twin        twins.Twin
		identifyErr error
		userID      string
	}{
		{
			desc:        "view existing twin",
			id:          twin.ID,
			auth:        token,
			status:      http.StatusOK,
			res:         twres,
			err:         nil,
			twin:        twin,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "view non-existent twin",
			id:          strconv.FormatUint(wrongID, 10),
			auth:        token,
			status:      http.StatusNotFound,
			res:         twinRes{},
			err:         svcerr.ErrNotFound,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "view twin by passing invalid token",
			id:          twin.ID,
			auth:        invalidtoken,
			status:      http.StatusForbidden,
			res:         twinRes{},
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "view twin by passing empty token",
			id:          twin.ID,
			auth:        "",
			status:      http.StatusUnauthorized,
			res:         twinRes{},
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("RetrieveByID", mock.Anything, tc.id).Return(tc.twin, tc.err)
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))

		var resData twinRes
		err = json.NewDecoder(res.Body).Decode(&resData)
		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error while decoding response body: %s\n", tc.desc, err))
		assert.Equal(t, tc.res, resData, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, resData))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestListTwins(t *testing.T) {
	svc, auth, twinRepo, _, _ := NewService()
	ts := newServer(svc)
	defer ts.Close()

	var data []twinRes
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("%s-%d", twinName, i)
		twin := twins.Twin{
			Owner:    email,
			Name:     name,
			ID:       testsutil.GenerateUUID(t),
			Revision: 150,
		}

		twres := twinRes{
			Owner:    twin.Owner,
			ID:       twin.ID,
			Name:     twin.Name,
			Revision: twin.Revision,
			Metadata: twin.Metadata,
		}
		data = append(data, twres)
	}

	baseURL := fmt.Sprintf("%s/twins", ts.URL)
	queryFmt := "%s?offset=%d&limit=%d"
	cases := []struct {
		desc        string
		auth        string
		status      int
		url         string
		res         []twinRes
		err         error
		page        twins.Page
		identifyErr error
		userID      string
	}{
		{
			desc:   "get a list of twins",
			auth:   token,
			status: http.StatusOK,
			url:    baseURL,
			res:    data[0:10],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[0:10]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with invalid token",
			auth:        invalidtoken,
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 1),
			res:         nil,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "get a list of twins with empty token",
			auth:        "",
			status:      http.StatusUnauthorized,
			url:         fmt.Sprintf(queryFmt, baseURL, 0, 1),
			res:         nil,
			err:         svcerr.ErrAuthentication,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:   "get a list of twins with valid offset and limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 25, 40),
			res:    data[25:65],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[25:65]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of twins with offset + limit > total",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf(queryFmt, baseURL, 91, 20),
			res:    data[91:],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[91:]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with negative offset",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, -1, 5),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with negative limit",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, 1, -5),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with zero limit",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf(queryFmt, baseURL, 1, 0),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with limit greater than max",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s?offset=%d&limit=%d", baseURL, 0, 110),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with invalid offset",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s%s", baseURL, "?offset=e&limit=5"),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with invalid limit",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s%s", baseURL, "?offset=5&limit=e"),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of twins without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", baseURL, 5),
			res:    data[0:5],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[0:5]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of twins without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", baseURL, 1),
			res:    data[1:11],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[1:11]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins with invalid number of parameters",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s%s", baseURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of twins with redundant query parameters",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", baseURL, 0, 5),
			res:    data[0:5],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[0:5]),
			},
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "get a list of twins filtering with invalid name",
			auth:        token,
			status:      http.StatusBadRequest,
			url:         fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", baseURL, 0, 5, invalidName),
			res:         nil,
			err:         svcerr.ErrMalformedEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:   "get a list of twins filtering with valid name",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", baseURL, 2, 1, twinName+"-2"),
			res:    data[2:3],
			err:    nil,
			page: twins.Page{
				Twins: convTwin(data[2:3]),
			},
			identifyErr: nil,
			userID:      validID,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: tc.userID}, nil)
		repoCall := twinRepo.On("RetrieveAll", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.page, tc.err)
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var resData twinsPageRes
		if tc.res != nil {
			err = json.NewDecoder(res.Body).Decode(&resData)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		}

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, resData.Twins, fmt.Sprintf("%s: got incorrect list of twins", tc.desc))
		authCall.Unset()
		repoCall.Unset()
	}
}

func TestRemoveTwin(t *testing.T) {
	svc, auth, twinRepo, twinCache, _ := NewService()
	ts := newServer(svc)
	defer ts.Close()

	twin := twins.Twin{
		ID: testsutil.GenerateUUID(t),
	}

	cases := []struct {
		desc        string
		id          string
		auth        string
		status      int
		err         error
		removeErr   error
		identifyErr error
		userID      string
	}{
		{
			desc:        "delete existing twin",
			id:          twin.ID,
			auth:        token,
			status:      http.StatusNoContent,
			err:         nil,
			removeErr:   nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "delete non-existent twin",
			id:          strconv.FormatUint(wrongID, 10),
			auth:        token,
			status:      http.StatusNoContent,
			err:         nil,
			removeErr:   nil,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "delete twin by passing empty id",
			id:          "",
			auth:        token,
			status:      http.StatusMethodNotAllowed,
			err:         svcerr.ErrMalformedEntity,
			removeErr:   svcerr.ErrRemoveEntity,
			identifyErr: nil,
			userID:      validID,
		},
		{
			desc:        "delete twin with invalid token",
			id:          twin.ID,
			auth:        invalidtoken,
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
			removeErr:   svcerr.ErrRemoveEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
		{
			desc:        "delete twin with empty token",
			id:          twin.ID,
			auth:        "",
			status:      http.StatusUnauthorized,
			err:         svcerr.ErrAuthentication,
			removeErr:   svcerr.ErrRemoveEntity,
			identifyErr: svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.auth}).Return(&magistrala.IdentityRes{Id: tc.userID}, tc.identifyErr)
		repoCall := twinRepo.On("Remove", mock.Anything, tc.id).Return(tc.removeErr)
		cacheCall2 := twinCache.On("Remove", mock.Anything, tc.id).Return(tc.err)
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		authCall.Unset()
		repoCall.Unset()
		cacheCall2.Unset()
	}
}

func convTwin(data []twinRes) []twins.Twin {
	twinSlice := make([]twins.Twin, len(data))
	for i, d := range data {
		twinSlice[i].ID = d.ID
		twinSlice[i].Name = d.Name
		twinSlice[i].Owner = d.Owner
		twinSlice[i].Revision = d.Revision
		twinSlice[i].Metadata = d.Metadata
	}
	return twinSlice
}
