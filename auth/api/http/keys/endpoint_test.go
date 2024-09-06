// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package keys_test

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

	"github.com/absmach/magistrala/auth"
	httpapi "github.com/absmach/magistrala/auth/api/http"
	"github.com/absmach/magistrala/auth/jwt"
	"github.com/absmach/magistrala/auth/mocks"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	policymocks "github.com/absmach/magistrala/pkg/policy/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	secret          = "secret"
	contentType     = "application/json"
	id              = "123e4567-e89b-12d3-a456-000000000001"
	email           = "user@example.com"
	loginDuration   = 30 * time.Minute
	refreshDuration = 24 * time.Hour
	invalidDuration = 7 * 24 * time.Hour
)

type issueRequest struct {
	Duration time.Duration `json:"duration,omitempty"`
	Type     uint32        `json:"type,omitempty"`
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

func newService() (auth.Service, *mocks.KeyRepository) {
	krepo := new(mocks.KeyRepository)
	prepo := new(mocks.PolicyAgent)
	drepo := new(mocks.DomainsRepository)
	idProvider := uuid.NewMock()
	policy := new(policymocks.PolicyClient)

	t := jwt.New([]byte(secret))

	return auth.New(krepo, drepo, idProvider, t, prepo, policy, loginDuration, refreshDuration, invalidDuration), krepo
}

func newServer(svc auth.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc, mglog.NewMock(), "")
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestIssue(t *testing.T) {
	svc, krepo := newService()
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	lk := issueRequest{Type: uint32(auth.AccessKey)}
	ak := issueRequest{Type: uint32(auth.APIKey), Duration: time.Hour}
	rk := issueRequest{Type: uint32(auth.RecoveryKey)}

	cases := []struct {
		desc   string
		req    string
		ct     string
		token  string
		status int
	}{
		{
			desc:   "issue login key with empty token",
			req:    toJSON(lk),
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue API key",
			req:    toJSON(ak),
			ct:     contentType,
			token:  token.AccessToken,
			status: http.StatusCreated,
		},
		{
			desc:   "issue recovery key",
			req:    toJSON(rk),
			ct:     contentType,
			token:  token.AccessToken,
			status: http.StatusCreated,
		},
		{
			desc:   "issue login key wrong content type",
			req:    toJSON(lk),
			ct:     "",
			token:  token.AccessToken,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue recovery key wrong content type",
			req:    toJSON(rk),
			ct:     "",
			token:  token.AccessToken,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue key with an invalid token",
			req:    toJSON(ak),
			ct:     contentType,
			token:  "wrong",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue recovery key with empty token",
			req:    toJSON(rk),
			ct:     contentType,
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue key with invalid request",
			req:    "{",
			ct:     contentType,
			token:  token.AccessToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON",
			req:    "{invalid}",
			ct:     contentType,
			token:  token.AccessToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON content",
			req:    `{"Type":{"key":"AccessToken"}}`,
			ct:     contentType,
			token:  token.AccessToken,
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      client,
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/keys", ts.URL),
			contentType: tc.ct,
			token:       tc.token,
			body:        strings.NewReader(tc.req),
		}
		repocall := krepo.On("Save", mock.Anything, mock.Anything).Return("", nil)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repocall.Unset()
	}
}

func TestRetrieve(t *testing.T) {
	svc, krepo := newService()
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), Subject: id}

	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	k, err := svc.Issue(context.Background(), token.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		key    auth.Key
		status int
		err    error
	}{
		{
			desc:  "retrieve an existing key",
			id:    k.AccessToken,
			token: token.AccessToken,
			key: auth.Key{
				Subject:   id,
				Type:      auth.AccessKey,
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(refreshDuration),
			},
			status: http.StatusOK,
			err:    nil,
		},
		{
			desc:   "retrieve a non-existing key",
			id:     "non-existing",
			token:  token.AccessToken,
			status: http.StatusBadRequest,
			err:    svcerr.ErrNotFound,
		},
		{
			desc:   "retrieve a key with an invalid token",
			id:     k.AccessToken,
			token:  "wrong",
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
		{
			desc:   "retrieve a key with an empty token",
			token:  "",
			id:     k.AccessToken,
			status: http.StatusUnauthorized,
			err:    svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/keys/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		repocall := krepo.On("Retrieve", mock.Anything, mock.Anything, mock.Anything).Return(tc.key, tc.err)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repocall.Unset()
	}
}

func TestRevoke(t *testing.T) {
	svc, krepo := newService()
	token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.AccessKey, IssuedAt: time.Now(), Subject: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), Subject: id}

	repocall := krepo.On("Save", mock.Anything, mock.Anything).Return(mock.Anything, nil)
	k, err := svc.Issue(context.Background(), token.AccessToken, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	repocall.Unset()

	ts := newServer(svc)
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
	}{
		{
			desc:   "revoke an existing key",
			id:     k.AccessToken,
			token:  token.AccessToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke a non-existing key",
			id:     "non-existing",
			token:  token.AccessToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke key with invalid token",
			id:     k.AccessToken,
			token:  "wrong",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "revoke key with empty token",
			id:     k.AccessToken,
			token:  "",
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodDelete,
			url:    fmt.Sprintf("%s/keys/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		repocall := krepo.On("Remove", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		repocall.Unset()
	}
}
