// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	token       = "token"
	wrongValue  = "wrong_value"
	wrongID     = 0
	maxNameSize = 1024
	nameKey     = "name"
	ascKey      = "asc"
	descKey     = "desc"
)

var (
	thing = things.Thing{
		Name:     "test_app",
		Metadata: map[string]interface{}{"test": "data"},
	}
	channel = things.Channel{
		Name:     "test",
		Metadata: map[string]interface{}{"test": "data"},
	}
	invalidName    = strings.Repeat("m", maxNameSize+1)
	notFoundRes    = toJSON(errorRes{things.ErrNotFound.Error()})
	unauthzRes     = toJSON(errorRes{things.ErrAuthorization.Error()})
	unauthRes      = toJSON(errorRes{things.ErrUnauthorizedAccess.Error()})
	searchThingReq = things.PageMetadata{
		Limit:  5,
		Offset: 0,
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
		req.Header.Set("Authorization", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func newService(tokens map[string]string) things.Service {
	policies := []mocks.MockSubjectSet{{Object: "users", Relation: "member"}}
	auth := mocks.NewAuthService(tokens, map[string][]mocks.MockSubjectSet{email: policies})
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, thingsRepo, channelsRepo, chanCache, thingCache, idProvider)
}

func newServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestCreateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	th := thing
	th.Key = "key"
	data := toJSON(th)

	th.Name = invalidName
	invalidData := toJSON(th)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "add valid thing",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/things/001",
		},
		{
			desc:        "add thing with existing key",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
			location:    "",
		},
		{
			desc:        "add thing with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/things/002",
		},
		{
			desc:        "add thing with invalid auth token",
			req:         data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "add thing with empty auth token",
			req:         data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "add thing with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add thing without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
		{
			desc:        "add thing with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))
	}
}

func TestCreateThings(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := `[{"name": "1", "key": "1"}, {"name": "2", "key": "2"}]`
	invalidData := fmt.Sprintf(`[{"name": "%s", "key": "10"}]`, invalidName)

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid things",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    "",
		},
		{
			desc:        "create things with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with invalid request format",
			data:        "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with invalid name",
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create things with empty JSON array",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create thing with existing key",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
			response:    "",
		},
		{
			desc:        "create thing with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create thing with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create thing without content type",
			data:        data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/bulk", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(thing)
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]

	th2 := thing
	th2.Name = invalidName
	invalidData := toJSON(th2)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing thing",
			req:         data,
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with empty JSON request",
			req:         "{}",
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existent thing",
			req:         data,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update thing with invalid id",
			req:         data,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th1.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th1.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         "",
			id:          th1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th1.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update thing with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestShareThing(t *testing.T) {
	token2 := "token2"
	svc := newService(map[string]string{token: email, token2: "user@ex.com"})
	ts := newServer(svc)
	defer ts.Close()

	type shareThingReq struct {
		UserIDs  []string `json:"user_ids"`
		Policies []string `json:"policies"`
	}

	data := toJSON(shareThingReq{UserIDs: []string{"token2"}, Policies: []string{"read"}})
	invalidData := toJSON(shareThingReq{})
	invalidPolicies := toJSON(shareThingReq{UserIDs: []string{"token2"}, Policies: []string{"wrong"}})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc        string
		req         string
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "share a thing",
			req:         data,
			thingID:     th.ID,
			contentType: contentType,
			token:       token,
			status:      http.StatusOK,
		},
		{
			desc:        "share a thing with empty content-type",
			req:         data,
			thingID:     th.ID,
			contentType: "",
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "share a thing with empty req body",
			req:         "",
			thingID:     th.ID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "share a thing with empty token",
			req:         data,
			thingID:     th.ID,
			contentType: contentType,
			token:       "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "share a thing with empty thing id",
			req:         data,
			thingID:     "",
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "share a thing with invalid req body",
			req:         invalidData,
			thingID:     th.ID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "share a thing with invalid policies request",
			req:         invalidPolicies,
			thingID:     th.ID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "share a thing with invalid token",
			req:         data,
			thingID:     th.ID,
			contentType: contentType,
			token:       "invalid",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "share a thing with unauthorized access",
			req:         data,
			thingID:     th.ID,
			contentType: contentType,
			token:       token2,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/share", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUpdateKey(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	th := thing
	th.Key = "key"
	ths, err := svc.CreateThings(context.Background(), token, th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th = ths[0]

	th.Key = "new-key"
	data := toJSON(th)

	th.Key = "key"
	dummyData := toJSON(th)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update key for an existing thing",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update thing with conflicting key",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusConflict,
		},
		{
			desc:        "update key with empty JSON request",
			req:         "{}",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update key of non-existent thing",
			req:         dummyData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update thing with invalid id",
			req:         dummyData,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update thing with invalid user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with empty user token",
			req:         data,
			id:          th.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update thing with invalid data format",
			req:         "{",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing with empty request",
			req:         "",
			id:          th.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update thing without content type",
			req:         data,
			id:          th.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/key", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	data := toJSON(thingRes{
		ID:       th.ID,
		Name:     th.Name,
		Key:      th.Key,
		Metadata: th.Metadata,
	})

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view existing thing",
			id:     th.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent thing",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusForbidden,
			res:    unauthzRes,
		},
		{
			desc:   "view thing by passing invalid token",
			id:     th.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view thing by passing empty token",
			id:     th.ID,
			auth:   "",
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view thing by passing invalid id",
			id:     "invalid",
			auth:   token,
			status: http.StatusForbidden,
			res:    unauthzRes,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
		})
	}

	thingURL := fmt.Sprintf("%s/things", ts.URL)
	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things ordered by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=desc", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things ordered by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=asc", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=wrong", thingURL, 0, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=wrong", thingURL, 0, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things with zero limit and offset 1",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 1, 0),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", thingURL, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", thingURL, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", thingURL, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", thingURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", thingURL, ""),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", thingURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", thingURL, 0, 5, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of things sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, "wrong", descKey),
			res:    nil,
		},
		{
			desc:   "get a list of things sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", thingURL, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestSearchThings(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	th := searchThingReq
	validData := toJSON(th)

	th.Dir = "desc"
	th.Order = "name"
	descData := toJSON(th)

	th.Dir = "asc"
	ascData := toJSON(th)

	th.Order = "wrong"
	invalidOrderData := toJSON(th)

	th = searchThingReq
	th.Dir = "wrong"
	invalidDirData := toJSON(th)

	th = searchThingReq
	th.Limit = 110
	limitMaxData := toJSON(th)

	th.Limit = 0
	zeroLimitData := toJSON(th)

	th = searchThingReq
	th.Name = invalidName
	invalidNameData := toJSON(th)

	th.Name = invalidName
	invalidData := toJSON(th)

	data := []thingRes{}
	for i := 0; i < 100; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		ths, err := svc.CreateThings(context.Background(), token, things.Thing{Name: name, Metadata: map[string]interface{}{"test": name}})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
		})
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		req    string
		res    []thingRes
	}{
		{
			desc:   "search things",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things ordered by name descendent",
			auth:   token,
			status: http.StatusOK,
			req:    descData,
			res:    data[0:5],
		},
		{
			desc:   "search things ordered by name ascendent",
			auth:   token,
			status: http.StatusOK,
			req:    ascData,
			res:    data[0:5],
		},
		{
			desc:   "search things with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
		{
			desc:   "search things with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things with invalid data",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidData,
			res:    nil,
		},
		{
			desc:   "search things with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			req:    validData,
			res:    nil,
		},
		{
			desc:   "search things with zero limit",
			auth:   token,
			status: http.StatusOK,
			req:    zeroLimitData,
			res:    data[0:10],
		},
		{
			desc:   "search things without offset",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			req:    limitMaxData,
			res:    nil,
		},
		{
			desc:   "search things with default URL",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things filtering with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidNameData,
			res:    nil,
		},
		{
			desc:   "search things sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			req:    validData,
			res:    data[0:5],
		},
		{
			desc:   "search things sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidOrderData,
			res:    nil,
		},
		{
			desc:   "search things sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			req:    invalidDirData,
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPost,
			url:    fmt.Sprintf("%s/things/search", ts.URL),
			token:  tc.auth,
			body:   strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestListThingsByChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	data := []thingRes{}
	for i := 0; i < 101; i++ {
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		data = append(data, thingRes{
			ID:       th.ID,
			Name:     th.Name,
			Key:      th.Key,
			Metadata: th.Metadata,
		})
	}
	thingURL := fmt.Sprintf("%s/channels", ts.URL)

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []thingRes
	}{
		{
			desc:   "get a list of things by channel",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, -5),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?limit=%d", thingURL, ch.ID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d", thingURL, ch.ID, 1),
			res:    data[1:11],
		},
		{
			desc:   "get a list of things by channel with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&value=something", thingURL, ch.ID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d", thingURL, ch.ID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things", thingURL, ch.ID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of things by channel with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things%s", thingURL, ch.ID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of things by channel sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of things by channel sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/things?offset=%d&limit=%d&order=%s&dir=%s", thingURL, ch.ID, 0, 5, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data thingsPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Things, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Things))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "delete existing thing",
			id:     th.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "delete non-existent thing",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusForbidden,
		},
		{
			desc:   "delete thing with invalid token",
			id:     th.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "delete thing with empty token",
			id:     th.ID,
			auth:   "",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/things/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := toJSON(channel)

	th := channel
	th.Name = invalidName
	invalidData := toJSON(th)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "create new channel",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/channels/001",
		},
		{
			desc:        "create new channel with invalid token",
			req:         data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "create new channel with empty token",
			req:         data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			location:    "",
		},
		{
			desc:        "create new channel with invalid data format",
			req:         "{",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "create new channel with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/channels/002",
		},
		{
			desc:        "create new channel with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "create new channel without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
		{
			desc:        "create new channel with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.location, location, fmt.Sprintf("%s: expected location %s got %s", tc.desc, tc.location, location))
	}
}

func TestCreateChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	data := `[{"name": "1"}, {"name": "2"}]`
	invalidData := fmt.Sprintf(`[{"name": "%s"}]`, invalidName)

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid channels",
			data:        data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    "",
		},
		{
			desc:        "create channel with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create channels with empty JSON",
			data:        "[]",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "create channel with invalid auth token",
			data:        data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "create channel with empty auth token",
			data:        data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:     "create channel with invalid request format",
			data:     "}",
			auth:     token,
			status:   http.StatusUnsupportedMediaType,
			response: "",
		},
		{
			desc:        "create channel without content type",
			data:        data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    "",
		},
		{
			desc:        "create channel with invalid name",
			data:        invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/channels/bulk", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	c := channel
	c.Name = "updated_channel"
	updateData := toJSON(c)

	c.Name = invalidName
	invalidData := toJSON(c)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing channel",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existing channel",
			req:         updateData,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update channel with invalid id",
			req:         updateData,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update channel with invalid token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update channel with empty token",
			req:         updateData,
			id:          ch.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update channel with invalid data format",
			req:         "}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with empty JSON object",
			req:         "{}",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update channel with empty request",
			req:         "",
			id:          ch.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update channel with missing content type",
			req:         updateData,
			id:          ch.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update channel with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sch := chs[0]

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	svc.Connect(context.Background(), token, []string{sch.ID}, []string{th.ID})

	data := toJSON(channelRes{
		ID:       sch.ID,
		Name:     sch.Name,
		Metadata: sch.Metadata,
	})

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view existing channel",
			id:     sch.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent channel",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    notFoundRes,
		},
		{
			desc:   "view channel with invalid token",
			id:     sch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view channel with empty token",
			id:     sch.ID,
			auth:   "",
			status: http.StatusUnauthorized,
			res:    unauthRes,
		},
		{
			desc:   "view channel with invalid id",
			id:     "invalid",
			auth:   token,
			status: http.StatusNotFound,
			res:    notFoundRes,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body := strings.Trim(string(data), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, body, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, body))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	channels := []channelRes{}
	for i := 0; i < 101; i++ {
		name := "name_" + fmt.Sprintf("%03d", i+1)
		chs, err := svc.CreateChannels(context.Background(), token,
			things.Channel{
				Name:     name,
				Metadata: map[string]interface{}{"test": "data"},
			})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		ch := chs[0]
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]
		svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})

		channels = append(channels, channelRes{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}
	channelURL := fmt.Sprintf("%s/channels", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []channelRes
	}{
		{
			desc:   "get a list of channels",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels ordered by id descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=id&dir=desc", channelURL, 0, 6),
			res:    channels[len(channels)-6:],
		},
		{
			desc:   "get a list of channels ordered by id ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=id&dir=asc", channelURL, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=wrong", channelURL, 0, 6),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid dir",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=name&dir=wrong", channelURL, 0, 6),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels with zero limit and offset 1",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 1, 0),
			res:    channels[1:11],
		},
		{
			desc:   "get a list of channels with no offset provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?limit=%d", channelURL, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with no limit provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d", channelURL, 1),
			res:    channels[1:11],
		},
		{
			desc:   "get a list of channels with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&value=something", channelURL, 0, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d", channelURL, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of channels with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s%s", channelURL, ""),
			res:    channels[0:10],
		},
		{
			desc:   "get a list of channels with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s%s", channelURL, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of channels with invalid name",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&name=%s", channelURL, 0, 10, invalidName),
			res:    nil,
		},
		{
			desc:   "get a list of channels sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, ascKey),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, descKey),
			res:    channels[len(channels)-6:],
		},
		{
			desc:   "get a list of channels sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of channels sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s?offset=%d&limit=%d&order=%s&dir=%s", channelURL, 0, 6, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body channelsPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Channels, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Channels))
	}
}

func TestListChannelsByThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	channels := []channelRes{}
	for i := 0; i < 101; i++ {
		chs, err := svc.CreateChannels(context.Background(), token, channel)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		ch := chs[0]
		err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

		channels = append(channels, channelRes{
			ID:       ch.ID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		})
	}
	channelURL := fmt.Sprintf("%s/things", ts.URL)

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []channelRes
	}{
		{
			desc:   "get a list of channels by thing",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, 0, 6),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels by thing with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with empty token",
			auth:   "",
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, -1, 5),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, 1, 0),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with no offset provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?limit=%d", channelURL, th.ID, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels by thing with no limit provided",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d", channelURL, th.ID, 1),
			res:    channels[1:11],
		},
		{
			desc:   "get a list of channels by thing with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d&value=something", channelURL, th.ID, 0, 5),
			res:    channels[0:5],
		},
		{
			desc:   "get a list of channels by thing with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d", channelURL, th.ID, 0, 110),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels", channelURL, th.ID),
			res:    channels[0:10],
		},
		{
			desc:   "get a list of channels by thing with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels%s", channelURL, th.ID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels%s", channelURL, th.ID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels%s", channelURL, th.ID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing sorted by name ascendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d&order=%s&dir=%s", channelURL, th.ID, 0, 6, nameKey, ascKey),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels by thing sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d&order=%s&dir=%s", channelURL, th.ID, 0, 6, nameKey, descKey),
			res:    channels[0:6],
		},
		{
			desc:   "get a list of channels by thing sorted with inalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d&order=%s&dir=%s", channelURL, th.ID, 0, 6, "wrong", ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of channels by thing sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/%s/channels?offset=%d&limit=%d&order=%s&dir=%s", channelURL, th.ID, 0, 6, nameKey, "wrong"),
			res:    nil,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var body channelsPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, body.Channels, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, body.Channels))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	chs, _ := svc.CreateChannels(context.Background(), token, channel)
	ch := chs[0]

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "remove channel with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove existing channel",
			id:     ch.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove removed channel",
			id:     ch.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "remove channel with invalid token",
			id:     ch.ID,
			auth:   wrongValue,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "remove channel with empty token",
			id:     ch.ID,
			auth:   "",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestConnect(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ths, _ := svc.CreateThings(context.Background(), token, thing)
	th1 := ths[0]
	chs, _ := svc.CreateChannels(context.Background(), token, channel)
	ch1 := chs[0]
	chs, _ = svc.CreateChannels(context.Background(), otherToken, channel)
	ch2 := chs[0]

	cases := []struct {
		desc    string
		chanID  string
		thingID string
		auth    string
		status  int
	}{
		{
			desc:    "connect existing thing to existing channel",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusOK,
		},
		{
			desc:    "connect existing thing to non-existent channel",
			chanID:  strconv.FormatUint(wrongID, 10),
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect non-existing thing to existing channel",
			chanID:  ch1.ID,
			thingID: strconv.FormatUint(wrongID, 10),
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect existing thing to channel with invalid id",
			chanID:  "invalid",
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect thing with invalid id to existing channel",
			chanID:  ch1.ID,
			thingID: "invalid",
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "connect existing thing to existing channel with invalid token",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    wrongValue,
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "connect existing thing to existing channel with empty token",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    "",
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "connect thing from owner to channel of other user",
			chanID:  ch2.ID,
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodPut,
			url:    fmt.Sprintf("%s/channels/%s/things/%s", ts.URL, tc.chanID, tc.thingID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestCreateConnections(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thIDs := []string{}
	for _, th := range ths {
		thIDs = append(thIDs, th.ID)
	}

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chIDs1 := []string{}
	for _, ch := range chs {
		chIDs1 = append(chIDs1, ch.ID)
	}
	chs, err = svc.CreateChannels(context.Background(), otherToken, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chIDs2 := []string{}
	for _, ch := range chs {
		chIDs2 = append(chIDs2, ch.ID)
	}

	cases := []struct {
		desc        string
		channelIDs  []string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "connect existing things to existing channels",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "connect existing things to non-existent channels",
			channelIDs:  []string{strconv.FormatUint(wrongID, 10)},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect non-existing things to existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to channel with invalid id",
			channelIDs:  []string{"invalid"},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect things with invalid id to existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{"invalid"},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect existing things to empty channel ids",
			channelIDs:  []string{""},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty things id to existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{""},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect existing things to existing channels with invalid token",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect existing things to existing channels with empty token",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "connect things from owner to channels of other user",
			channelIDs:  chIDs2,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "connect with invalid content type",
			channelIDs:  chIDs2,
			thingIDs:    thIDs,
			auth:        token,
			contentType: "invalid",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "connect with invalid JSON",
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
			body:        "{",
		},
		{
			desc:        "connect valid thing ids with empty channel ids",
			channelIDs:  []string{},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect valid channel ids with empty thing ids",
			channelIDs:  chIDs1,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "connect empty channel ids and empty thing ids",
			channelIDs:  []string{},
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ChannelIDs []string `json:"channel_ids"`
			ThingIDs   []string `json:"thing_ids"`
		}{
			tc.channelIDs,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != "" {
			body = tc.body
		}

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/connect", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(body),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDisconnectList(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thIDs := []string{}
	for _, th := range ths {
		thIDs = append(thIDs, th.ID)
	}

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chIDs1 := []string{}
	for _, ch := range chs {
		chIDs1 = append(chIDs1, ch.ID)
	}

	chs, err = svc.CreateChannels(context.Background(), otherToken, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chIDs2 := []string{}
	for _, ch := range chs {
		chIDs2 = append(chIDs2, ch.ID)
	}

	err = svc.Connect(context.Background(), token, chIDs1, thIDs)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc        string
		channelIDs  []string
		thingIDs    []string
		auth        string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "disconnect existing things from existing channels",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "disconnect existing things from non-existent channels",
			channelIDs:  []string{strconv.FormatUint(wrongID, 10)},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect non-existing things from existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{strconv.FormatUint(wrongID, 10)},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from channel with invalid id",
			channelIDs:  []string{"invalid"},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect things with invalid id from existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{"invalid"},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect existing things from empty channel ids",
			channelIDs:  []string{""},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty things id from existing channels",
			channelIDs:  chIDs1,
			thingIDs:    []string{""},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect existing things from existing channels with invalid token",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        wrongValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect existing things from existing channels with empty token",
			channelIDs:  chIDs1,
			thingIDs:    thIDs,
			auth:        "",
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "disconnect things from channels of other user",
			channelIDs:  chIDs2,
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "disconnect with invalid content type",
			channelIDs:  chIDs2,
			thingIDs:    thIDs,
			auth:        token,
			contentType: "invalid",
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "disconnect with invalid JSON",
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
			body:        "{",
		},
		{
			desc:        "disconnect valid thing ids from empty channel ids",
			channelIDs:  []string{},
			thingIDs:    thIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from valid channel ids",
			channelIDs:  chIDs1,
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "disconnect empty thing ids from empty channel ids",
			channelIDs:  []string{},
			thingIDs:    []string{},
			auth:        token,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		data := struct {
			ChannelIDs []string `json:"channel_ids"`
			ThingIDs   []string `json:"thing_ids"`
		}{
			tc.channelIDs,
			tc.thingIDs,
		}
		body := toJSON(data)

		if tc.body != "" {
			body = tc.body
		}

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/disconnect", ts.URL),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDisconnnect(t *testing.T) {
	otherToken := "other_token"
	otherEmail := "other_user@example.com"
	svc := newService(map[string]string{
		token:      email,
		otherToken: otherEmail,
	})
	ts := newServer(svc)
	defer ts.Close()

	ths, _ := svc.CreateThings(context.Background(), token, thing)
	th1 := ths[0]
	chs, _ := svc.CreateChannels(context.Background(), token, channel)
	ch1 := chs[0]
	svc.Connect(context.Background(), token, []string{ch1.ID}, []string{th1.ID})
	chs, _ = svc.CreateChannels(context.Background(), otherToken, channel)
	ch2 := chs[0]

	cases := []struct {
		desc    string
		chanID  string
		thingID string
		auth    string
		status  int
	}{
		{
			desc:    "disconnect connected thing from channel",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNoContent,
		},
		{
			desc:    "disconnect non-connected thing from channel",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect non-existent thing from channel",
			chanID:  ch1.ID,
			thingID: strconv.FormatUint(wrongID, 10),
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing from non-existent channel",
			chanID:  strconv.FormatUint(wrongID, 10),
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing from channel with invalid token",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    wrongValue,
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "disconnect thing from channel with empty token",
			chanID:  ch1.ID,
			thingID: th1.ID,
			auth:    "",
			status:  http.StatusUnauthorized,
		},
		{
			desc:    "disconnect owner's thing from someone elses channel",
			chanID:  ch2.ID,
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing with invalid id from channel",
			chanID:  ch1.ID,
			thingID: "invalid",
			auth:    token,
			status:  http.StatusNotFound,
		},
		{
			desc:    "disconnect thing from channel with invalid id",
			chanID:  "invalid",
			thingID: th1.ID,
			auth:    token,
			status:  http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/channels/%s/things/%s", ts.URL, tc.chanID, tc.thingID),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type thingRes struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type channelRes struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type thingsPageRes struct {
	Things []thingRes `json:"things"`
	Total  uint64     `json:"total"`
	Offset uint64     `json:"offset"`
	Limit  uint64     `json:"limit"`
}

type channelsPageRes struct {
	Channels []channelRes `json:"channels"`
	Total    uint64       `json:"total"`
	Offset   uint64       `json:"offset"`
	Limit    uint64       `json:"limit"`
}

type errorRes struct {
	Err string `json:"error"`
}
