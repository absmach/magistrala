// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/auth/http"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	token       = "token"
	wrong       = "wrong_value"
)

var (
	thing = things.Thing{
		Name:     "test_app",
		Metadata: map[string]interface{}{"test": "data"},
	}
	channel = things.Channel{
		Name:     "test_chan",
		Metadata: map[string]interface{}{"test": "data"},
	}
)

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
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
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func TestIdentify(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	ir := identifyReq{Token: th.Key}
	data := toJSON(ir)

	nonexistentData := toJSON(identifyReq{Token: wrong})

	cases := []struct {
		desc        string
		contentType string
		req         string
		statusCode  int
	}{
		{
			desc:        "identify existing thing",
			contentType: contentType,
			req:         data,
			statusCode:  http.StatusOK,
		},
		{
			desc:        "identify non-existent thing",
			contentType: contentType,
			req:         nonexistentData,
			statusCode:  http.StatusNotFound,
		},
		{
			desc:        "identify with missing content type",
			contentType: wrong,
			req:         data,
			statusCode:  http.StatusUnsupportedMediaType,
		},
		{
			desc:        "identify with empty JSON request",
			contentType: contentType,
			req:         "{}",
			statusCode:  http.StatusUnauthorized,
		},
		{
			desc:        "identify with invalid JSON request",
			contentType: contentType,
			req:         "",
			statusCode:  http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/identify", ts.URL),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.statusCode, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.statusCode, res.StatusCode))
	}
}

func TestCanAccessByKey(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("failed to create channel: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("failed to connect thing and channel: %s", err))

	data := toJSON(canAccessByKeyReq{
		Token: th.Key,
	})

	cases := []struct {
		desc        string
		contentType string
		chanID      string
		req         string
		statusCode  int
	}{
		{
			desc:        "check access for connected thing and channel",
			contentType: contentType,
			chanID:      ch.ID,
			req:         data,
			statusCode:  http.StatusOK,
		},
		{
			desc:        "check access for not connected thing and channel",
			contentType: contentType,
			chanID:      wrong,
			req:         data,
			statusCode:  http.StatusForbidden,
		},
		{
			desc:        "check access with invalid content type",
			contentType: wrong,
			chanID:      ch.ID,
			req:         data,
			statusCode:  http.StatusUnsupportedMediaType,
		},
		{
			desc:        "check access with empty JSON request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "{}",
			statusCode:  http.StatusUnauthorized,
		},
		{
			desc:        "check access with invalid JSON request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "}",
			statusCode:  http.StatusBadRequest,
		},
		{
			desc:        "check access with empty request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "",
			statusCode:  http.StatusBadRequest,
		},
		{
			desc:        "check access with empty channel id",
			contentType: contentType,
			chanID:      "",
			req:         data,
			statusCode:  http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/identify/channels/%s/access-by-key", ts.URL, tc.chanID),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.statusCode, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.statusCode, res.StatusCode))
	}
}

func TestCanAccessByID(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("failed to create thing: %s", err))
	th := ths[0]

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("failed to create channel: %s", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("failed to connect thing and channel: %s", err))

	data := toJSON(canAccessByIDReq{
		ThingID: th.ID,
	})

	cases := []struct {
		desc        string
		contentType string
		chanID      string
		req         string
		statusCode  int
	}{
		{
			desc:        "check access for connected thing and channel",
			contentType: contentType,
			chanID:      ch.ID,
			req:         data,
			statusCode:  http.StatusOK,
		},
		{
			desc:        "check access for not connected thing and channel",
			contentType: contentType,
			chanID:      wrong,
			req:         data,
			statusCode:  http.StatusForbidden,
		},
		{
			desc:        "check access with invalid content type",
			contentType: wrong,
			chanID:      ch.ID,
			req:         data,
			statusCode:  http.StatusUnsupportedMediaType,
		},
		{
			desc:        "check access with empty JSON request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "{}",
			statusCode:  http.StatusBadRequest,
		},
		{
			desc:        "check access with invalid JSON request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "}",
			statusCode:  http.StatusBadRequest,
		},
		{
			desc:        "check access with empty request",
			contentType: contentType,
			chanID:      ch.ID,
			req:         "",
			statusCode:  http.StatusBadRequest,
		},
		{
			desc:        "check access with empty channel id",
			contentType: contentType,
			chanID:      "",
			req:         data,
			statusCode:  http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/identify/channels/%s/access-by-id", ts.URL, tc.chanID),
			contentType: tc.contentType,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.statusCode, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.statusCode, res.StatusCode))
	}
}

type identifyReq struct {
	Token string `json:"token"`
}

type canAccessByKeyReq struct {
	Token string `json:"token"`
}

type canAccessByIDReq struct {
	ThingID string `json:"thing_id"`
}
