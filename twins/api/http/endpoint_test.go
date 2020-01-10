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

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/mainflux/mainflux/twins"
	httpapi "github.com/mainflux/mainflux/twins/api/http"
	"github.com/mainflux/mainflux/twins/mocks"
	twmqtt "github.com/mainflux/mainflux/twins/mqtt"
	nats "github.com/nats-io/go-nats"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	token       = "token"
	wrongValue  = "wrong_value"
	thingID     = "5b68df78-86f7-48a6-ac4f-bb24dd75c39e"
	wrongID     = 0
	maxNameSize = 1024
	natsURL     = "nats://localhost:4222"
	topic       = "topic"
)

var invalidName = strings.Repeat("m", maxNameSize+1)

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

func newService(tokens map[string]string) twins.Service {
	auth := mocks.NewAuthNServiceClient(tokens)
	twinsRepo := mocks.NewTwinRepository()
	statesRepo := mocks.NewStateRepository()
	idp := mocks.NewIdentityProvider()

	nc, _ := nats.Connect(natsURL)

	opts := mqtt.NewClientOptions()
	pc := mqtt.NewClient(opts)

	mc := twmqtt.New(pc, topic)

	return twins.New(nc, mc, auth, twinsRepo, statesRepo, idp)
}

func newServer(svc twins.Service) *httptest.Server {
	mux := httpapi.MakeHandler(mocktracer.New(), svc)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func TestAddTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	// tw := twins.Twin{ThingID: thingID}
	tw := twinReq{ThingID: thingID}
	data := toJSON(tw)

	tw.Name = invalidName
	invalidData := toJSON(tw)

	cases := []struct {
		desc        string
		req         string
		contentType string
		auth        string
		status      int
		location    string
	}{
		{
			desc:        "add valid twin",
			req:         data,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			location:    "/twins/123e4567-e89b-12d3-a456-000000000001",
		},
		{
			desc:        "add twin with empty JSON request",
			req:         "{}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add twin with invalid auth token",
			req:         data,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
			location:    "",
		},
		{
			desc:        "add twin with empty auth token",
			req:         data,
			contentType: contentType,
			auth:        "",
			status:      http.StatusForbidden,
			location:    "",
		},
		{
			desc:        "add twin with invalid request format",
			req:         "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add twin with empty request",
			req:         "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			location:    "",
		},
		{
			desc:        "add twin without content type",
			req:         data,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			location:    "",
		},
		{
			desc:        "add twin with invalid name",
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
	}
}

func TestUpdateTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	twin := twins.Twin{ThingID: thingID}
	def := twins.Definition{}
	data := toJSON(twin)
	stw, _ := svc.AddTwin(context.Background(), token, twin, def)

	tw := twin
	tw.Name = invalidName
	invalidData := toJSON(tw)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing twin",
			req:         data,
			id:          stw.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update twin with empty JSON request",
			req:         "{}",
			id:          stw.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existent twin",
			req:         data,
			id:          strconv.FormatUint(wrongID, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update twin with invalid user token",
			req:         data,
			id:          stw.ID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
		},
		{
			desc:        "update twin with empty user token",
			req:         data,
			id:          stw.ID,
			contentType: contentType,
			auth:        "",
			status:      http.StatusForbidden,
		},
		{
			desc:        "update twin with invalid data format",
			req:         "{",
			id:          stw.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update twin with empty request",
			req:         "",
			id:          stw.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update twin without content type",
			req:         data,
			id:          stw.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update twin with invalid name",
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
			url:         fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	def := twins.Definition{}
	twin := twins.Twin{ThingID: thingID}
	stw, err := svc.AddTwin(context.Background(), token, twin, def)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	twres := twinRes{
		Owner:       stw.Owner,
		Name:        stw.Name,
		ID:          stw.ID,
		ThingID:     stw.ThingID,
		Revision:    stw.Revision,
		Created:     stw.Created,
		Updated:     stw.Updated,
		Definitions: stw.Definitions,
		Metadata:    stw.Metadata,
	}
	data := toJSON(twres)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view existing twin",
			id:     stw.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view non-existent twin",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNotFound,
			res:    "",
		},
		{
			desc:   "view twin by passing invalid token",
			id:     stw.ID,
			auth:   wrongValue,
			status: http.StatusForbidden,
			res:    "",
		},
		{
			desc:   "view twin by passing empty id",
			id:     "",
			auth:   token,
			status: http.StatusBadRequest,
			res:    "",
		},
		{
			desc:   "view twin by passing empty token",
			id:     stw.ID,
			auth:   "",
			status: http.StatusForbidden,
			res:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		body, err := ioutil.ReadAll(res.Body)
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestRemoveTwin(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	def := twins.Definition{}
	twin := twins.Twin{ThingID: thingID}
	stw, _ := svc.AddTwin(context.Background(), token, twin, def)

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
	}{
		{
			desc:   "delete existing twin",
			id:     stw.ID,
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "delete non-existent twin",
			id:     strconv.FormatUint(wrongID, 10),
			auth:   token,
			status: http.StatusNoContent,
		},
		{
			desc:   "delete twin by passing empty id",
			id:     "",
			auth:   token,
			status: http.StatusBadRequest,
		},
		{
			desc:   "delete twin with invalid token",
			id:     stw.ID,
			auth:   wrongValue,
			status: http.StatusForbidden,
		},
		{
			desc:   "delete twin with empty token",
			id:     stw.ID,
			auth:   "",
			status: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/twins/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

type twinReq struct {
	token      string
	Name       string                 `json:"name,omitempty"`
	ThingID    string                 `json:"thing_id"`
	Definition twins.Definition       `json:"definition,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type twinRes struct {
	Owner       string                 `json:"owner"`
	Name        string                 `json:"name,omitempty"`
	ID          string                 `json:"id"`
	ThingID     string                 `json:"thing_id"`
	Revision    int                    `json:"revision"`
	Created     time.Time              `json:"created"`
	Updated     time.Time              `json:"updated"`
	Definitions []twins.Definition     `json:"definitions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
