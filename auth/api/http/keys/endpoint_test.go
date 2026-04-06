// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package keys_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/auth"
	httpapi "github.com/absmach/magistrala/auth/api/http"
	"github.com/absmach/magistrala/auth/mocks"
	mglog "github.com/absmach/magistrala/logger"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	contentType     = "application/json"
	id              = "123e4567-e89b-12d3-a456-000000000001"
	refreshDuration = 24 * time.Hour
	accessToken     = "valid token"
)

var Token = auth.Token{
	AccessToken: accessToken,
}

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

func newServer() (*httptest.Server, *mocks.Service) {
	svc := new(mocks.Service)
	mux := httpapi.MakeHandler(svc, mglog.NewMock(), "", 900, 60)

	return httptest.NewServer(mux), svc
}

func toJSON(data any) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func TestIssue(t *testing.T) {
	ts, svc := newServer()
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
		svcRes auth.Token
		svcErr error
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
			token:  accessToken,
			status: http.StatusCreated,
			svcRes: Token,
		},
		{
			desc:   "issue recovery key",
			req:    toJSON(rk),
			ct:     contentType,
			token:  accessToken,
			status: http.StatusCreated,
			svcRes: Token,
		},
		{
			desc:   "issue login key wrong content type",
			req:    toJSON(lk),
			ct:     "",
			token:  accessToken,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue recovery key wrong content type",
			req:    toJSON(rk),
			ct:     "",
			token:  accessToken,
			status: http.StatusUnsupportedMediaType,
		},
		{
			desc:   "issue key with an invalid token",
			req:    toJSON(ak),
			ct:     contentType,
			token:  "wrong",
			svcErr: svcerr.ErrAuthentication,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue recovery key with empty token",
			req:    toJSON(rk),
			ct:     contentType,
			token:  "",
			svcErr: svcerr.ErrAuthentication,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "issue key with invalid request",
			req:    "{",
			ct:     contentType,
			token:  accessToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON",
			req:    "{invalid}",
			ct:     contentType,
			token:  accessToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "issue key with invalid JSON content",
			req:    `{"Type":{"key":"AccessToken"}}`,
			ct:     contentType,
			token:  accessToken,
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
		svcCall := svc.On("Issue", mock.Anything, tc.token, mock.Anything).Return(tc.svcRes, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestRetrieve(t *testing.T) {
	ts, svc := newServer()
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		key    auth.Key
		status int
		svcRes auth.Key
		svcErr error
	}{
		{
			desc:  "retrieve an existing key",
			id:    id,
			token: accessToken,
			key: auth.Key{
				Subject:   id,
				Type:      auth.AccessKey,
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(refreshDuration),
			},
			status: http.StatusOK,
			svcErr: nil,
		},
		{
			desc:   "retrieve a non-existing key",
			id:     "non-existing",
			token:  accessToken,
			status: http.StatusNotFound,
			svcErr: svcerr.ErrNotFound,
		},
		{
			desc:   "retrieve a key with an invalid token",
			id:     accessToken,
			token:  "wrong",
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
		},
		{
			desc:   "retrieve a key with an empty token",
			token:  "",
			id:     accessToken,
			status: http.StatusUnauthorized,
			svcErr: svcerr.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/keys/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		svcCall := svc.On("RetrieveKey", mock.Anything, tc.token, tc.id).Return(tc.svcRes, tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestRevoke(t *testing.T) {
	ts, svc := newServer()
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		id     string
		token  string
		status int
		svcErr error
	}{
		{
			desc:   "revoke an existing key",
			id:     id,
			token:  accessToken,
			status: http.StatusNoContent,
		},
		{
			desc:   "revoke a non-existing key",
			id:     "non-existing",
			token:  accessToken,
			svcErr: svcerr.ErrNotFound,
			status: http.StatusNotFound,
		},
		{
			desc:   "revoke key with invalid token",
			id:     id,
			token:  "wrong",
			svcErr: svcerr.ErrAuthentication,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "revoke key with empty token",
			id:     id,
			token:  "",
			svcErr: svcerr.ErrAuthentication,
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
		svcCall := svc.On("Revoke", mock.Anything, tc.token, tc.id).Return(tc.svcErr)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func TestRetrieveJWKS(t *testing.T) {
	ts, svc := newServer()
	defer ts.Close()
	client := ts.Client()

	cases := []struct {
		desc   string
		svcRes []auth.PublicKeyInfo
		status int
	}{
		{
			desc:   "retrieve JWKS with keys",
			svcRes: []auth.PublicKeyInfo{newPublicKeyInfo(), newPublicKeyInfo()},
			status: http.StatusOK,
		},
		{
			desc:   "retrieve empty JWKS",
			svcRes: []auth.PublicKeyInfo{},
			status: http.StatusOK,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: client,
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/keys/.well-known/jwks.json", ts.URL),
		}
		svcCall := svc.On("RetrieveJWKS").Return(tc.svcRes)
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		svcCall.Unset()
	}
}

func newPublicKeyInfo() auth.PublicKeyInfo {
	return auth.PublicKeyInfo{
		KeyID:     "test-key-id",
		KeyType:   "OKP",
		Algorithm: "EdDSA",
		Use:       "sig",
		Curve:     "Ed25519",
		X:         "base64url-encoded-public-key",
	}
}
