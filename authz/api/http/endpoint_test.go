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

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/mainflux/mainflux/authz"
	httpapi "github.com/mainflux/mainflux/authz/api/http"
	"github.com/mainflux/mainflux/users/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contentType = "application/json"
	email       = "user@example.com"
	token       = "token"
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

	req.Header.Set("Referer", "http://localhost")
	return tr.client.Do(req)
}

func newService(tokens map[string]string) authz.Service {
	m := model.NewModel()
	m.AddDef("r", "r", "sub, obj, act")
	m.AddDef("p", "p", "sub, obj, act")
	m.AddDef("g", "g", "_, _")
	m.AddDef("e", "e", "some(where (p.eft == allow))")
	m.AddDef("m", "m", "( g(r.sub, p.sub) && keyMatch(r.obj, p.obj) && regexMatch(r.act, p.act) ) || r.sub == 'admin@example.com'")
	e, _ := casbin.NewSyncedEnforcer(m)

	auth := mocks.NewAuthService(tokens)

	return authz.New(e, auth)
}

func TestRemovePolicy(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	policy := authz.Policy{
		Subject: "admin",
		Object:  "users",
		Action:  "create",
	}

	_, err := svc.AddPolicy(context.Background(), token, policy)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "delete policy",
			data:        toJSON(policy),
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    "",
		},
		{
			desc:        "delete policy with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "delete policy with invalid request format",
			data:        "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "delete policy with empty auth token",
			data:        toJSON(policy),
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "delete policy without empty content type",
			data:        toJSON(policy),
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodDelete,
			url:         fmt.Sprintf("%s/policies", ts.URL),
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

func TestAddPolicy(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ts := newServer(svc)
	defer ts.Close()

	policy := `{"subject":"admin@email.com","object":"users","action":"create"}`

	cases := []struct {
		desc        string
		data        string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "add policy",
			data:        policy,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    "",
		},
		{
			desc:        "add policy with empty request",
			data:        "",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "add policy with invalid request format",
			data:        "}",
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    "",
		},
		{
			desc:        "add policy with empty auth token",
			data:        policy,
			contentType: contentType,
			auth:        "",
			status:      http.StatusUnauthorized,
			response:    "",
		},
		{
			desc:        "add policy without content type",
			data:        policy,
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
			url:         fmt.Sprintf("%s/policies", ts.URL),
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

func newServer(svc authz.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc, mocktracer.New())
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}
