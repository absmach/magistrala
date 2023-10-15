// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package policies_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mainflux/mainflux/auth"
	httpapi "github.com/mainflux/mainflux/auth/api/http"
	"github.com/mainflux/mainflux/auth/jwt"
	"github.com/mainflux/mainflux/auth/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

const (
	secret          = "secret"
	contentType     = "application/json"
	id              = uuid.Prefix + "-000000000001"
	email           = "user@example.com"
	unauthzID       = uuid.Prefix + "-000000000002"
	unauthzEmail    = "unauthz@example.com"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
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

func newService() auth.Service {
	krepo := new(mocks.Keys)
	prepo := new(mocks.PolicyAgent)
	idProvider := uuid.NewMock()

	t := jwt.New([]byte(secret))

	return auth.New(krepo, idProvider, t, prepo, loginDuration, refreshDuration)
}

func newServer(svc auth.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(svc, logger, "")
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

type addPolicyRequest struct {
	SubjectIDs []string `json:"subjects"`
	Policies   []string `json:"policies"`
	Object     string   `json:"object"`
}

func TestAddPolicies(t *testing.T) {
	svc := newService()
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	userLoginToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: unauthzID})
	assert.Nil(t, err, fmt.Sprintf("Issuing unauthorized user's key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	valid := addPolicyRequest{Object: "obj", Policies: []string{"read"}, SubjectIDs: []string{"user1", "user2"}}
	multipleValid := addPolicyRequest{Object: "obj", Policies: []string{"write", "delete"}, SubjectIDs: []string{"user1", "user2"}}
	invalidObject := addPolicyRequest{Object: "", Policies: []string{"read"}, SubjectIDs: []string{"user1", "user2"}}
	invalidPolicies := addPolicyRequest{Object: "obj", Policies: []string{"read", "invalid"}, SubjectIDs: []string{"user1", "user2"}}
	invalidSubjects := addPolicyRequest{Object: "obj", Policies: []string{"read", "access"}, SubjectIDs: []string{"", "user2"}}

	cases := []struct {
		desc   string
		token  string
		ct     string
		status int
		req    string
	}{
		{
			desc:   "Add policies with authorized access",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusCreated,
			req:    toJSON(valid),
		},
		{
			desc:   "Add multiple policies to multiple user",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusCreated,
			req:    toJSON(multipleValid),
		},
		{
			desc:   "Add policies with unauthorized access",
			token:  userLoginToken.AccessToken,
			ct:     contentType,
			status: http.StatusForbidden,
			req:    toJSON(valid),
		},
		{
			desc:   "Add policies with invalid token",
			token:  "invalid",
			ct:     contentType,
			status: http.StatusUnauthorized,
			req:    toJSON(valid),
		},
		{
			desc:   "Add policies with empty token",
			token:  "",
			ct:     contentType,
			status: http.StatusUnauthorized,
			req:    toJSON(valid),
		},
		{
			desc:   "Add policies with invalid content type",
			token:  token.AccessToken,
			ct:     "text/html",
			status: http.StatusUnsupportedMediaType,
			req:    toJSON(valid),
		},
		{
			desc:   "Add policies with empty content type",
			token:  token.AccessToken,
			ct:     "",
			status: http.StatusUnsupportedMediaType,
			req:    toJSON(valid),
		},
		{
			desc:   "Add policies with invalid object field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidObject),
		},
		{
			desc:   "Add policies with invalid policies field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidPolicies),
		},
		{
			desc:   "Add policies with invalid subjects field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidSubjects),
		},
		{
			desc:   "Add policies with empty request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/policies", ts.URL),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestDeletePolicies(t *testing.T) {
	svc := newService()
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing user key expected to succeed: %s", err))

	userLoginToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: unauthzID})
	assert.Nil(t, err, fmt.Sprintf("Issuing unauthorized user's key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	policies := addPolicyRequest{Object: "obj", Policies: []string{"read", "write", "delete"}, SubjectIDs: []string{"user1", "user2", "user3"}}
	err = svc.AddPolicies(context.Background(), token.AccessToken, policies.Object, policies.SubjectIDs, policies.Policies)
	assert.Nil(t, err, fmt.Sprintf("Adding policies expected to succeed: %s", err))

	validSingleDeleteReq := addPolicyRequest{Object: "obj", Policies: []string{"read"}, SubjectIDs: []string{"user1"}}
	validMultipleDeleteReq := addPolicyRequest{Object: "obj", Policies: []string{"write", "delete"}, SubjectIDs: []string{"user2", "user3"}}
	invalidObject := addPolicyRequest{Object: "", Policies: []string{"read"}, SubjectIDs: []string{"user1", "user2"}}
	invalidPolicies := addPolicyRequest{Object: "obj", Policies: []string{"read", "invalid"}, SubjectIDs: []string{"user1", "user2"}}
	invalidSubjects := addPolicyRequest{Object: "obj", Policies: []string{"read", "access"}, SubjectIDs: []string{"", "user2"}}

	cases := []struct {
		desc   string
		token  string
		ct     string
		req    string
		status int
	}{
		{
			desc:   "Delete policies with unauthorized access",
			token:  userLoginToken.AccessToken,
			ct:     contentType,
			status: http.StatusForbidden,
			req:    toJSON(validMultipleDeleteReq),
		},
		{
			desc:   "Delete policies with invalid token",
			token:  "invalid",
			ct:     contentType,
			status: http.StatusUnauthorized,
			req:    toJSON(validSingleDeleteReq),
		},
		{
			desc:   "Delete policies with empty token",
			token:  "",
			ct:     contentType,
			status: http.StatusUnauthorized,
			req:    toJSON(validSingleDeleteReq),
		},
		{
			desc:   "Delete policies with authorized access",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusNoContent,
			req:    toJSON(validSingleDeleteReq),
		},
		{
			desc:   "Delete multiple policies to multiple user",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusNoContent,
			req:    toJSON(validMultipleDeleteReq),
		},
		{
			desc:   "Delete policies with invalid content type",
			token:  token.AccessToken,
			ct:     "text/html",
			status: http.StatusUnsupportedMediaType,
			req:    toJSON(validMultipleDeleteReq),
		},
		{
			desc:   "Delete policies with empty content type",
			token:  token.AccessToken,
			ct:     "",
			status: http.StatusUnsupportedMediaType,
			req:    toJSON(validMultipleDeleteReq),
		},
		{
			desc:   "Delete policies with invalid object field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidObject),
		},
		{
			desc:   "Delete policies with invalid policies field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidPolicies),
		},
		{
			desc:   "Delete policies with invalid subjects field in request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    toJSON(invalidSubjects),
		},
		{
			desc:   "Delete policies with empty request body",
			token:  token.AccessToken,
			ct:     contentType,
			status: http.StatusBadRequest,
			req:    "",
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/policies", ts.URL),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
