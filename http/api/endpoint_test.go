// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	server "github.com/absmach/magistrala/http"
	"github.com/absmach/magistrala/http/api"
	"github.com/absmach/magistrala/http/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	mproxy "github.com/absmach/mproxy/pkg/http"
	"github.com/absmach/mproxy/pkg/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"

func newService(auth magistrala.AuthzServiceClient) session.Handler {
	pub := mocks.NewPublisher()
	return server.NewHandler(pub, mglog.NewMock(), auth)
}

func newTargetHTTPServer() *httptest.Server {
	mux := api.MakeHandler(instanceID)
	return httptest.NewServer(mux)
}

func newProxyHTPPServer(svc session.Handler, targetServer *httptest.Server) (*httptest.Server, error) {
	mp, err := mproxy.NewProxy("", targetServer.URL, svc, mglog.NewMock())
	if err != nil {
		return nil, err
	}
	return httptest.NewServer(http.HandlerFunc(mp.Handler)), nil
}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
	basicAuth   bool
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.token)
	}
	if tr.basicAuth && tr.token != "" {
		req.SetBasicAuth("", tr.token)
	}
	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func TestPublish(t *testing.T) {
	auth := new(authmocks.Service)
	chanID := "1"
	ctSenmlJSON := "application/senml+json"
	ctSenmlCBOR := "application/senml+cbor"
	ctJSON := "application/json"
	thingKey := "thing_key"
	invalidKey := authmocks.InvalidValue
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	msgJSON := `{"field1":"val1","field2":"val2"}`
	msgCBOR := `81A3616E6763757272656E746174206176FB3FF999999999999A`
	svc := newService(auth)
	target := newTargetHTTPServer()
	defer target.Close()
	ts, err := newProxyHTPPServer(svc, target)
	assert.Nil(t, err, fmt.Sprintf("failed to create proxy server with err: %v", err))

	defer ts.Close()

	auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{Subject: thingKey, Object: chanID, Domain: "", SubjectType: "thing", Permission: "publish", ObjectType: "group"}).Return(&magistrala.AuthorizeRes{Authorized: true, Id: ""}, nil)
	auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false, Id: ""}, nil)

	cases := map[string]struct {
		chanID      string
		msg         string
		contentType string
		key         string
		status      int
		basicAuth   bool
	}{
		"publish message": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/senml+cbor content-type": {
			chanID:      chanID,
			msg:         msgCBOR,
			contentType: ctSenmlCBOR,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with application/json content-type": {
			chanID:      chanID,
			msg:         msgJSON,
			contentType: ctJSON,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with empty key": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         "",
			status:      http.StatusBadGateway,
		},
		"publish message with basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			basicAuth:   true,
			status:      http.StatusAccepted,
		},
		"publish message with invalid key": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			status:      http.StatusBadRequest,
		},
		"publish message with invalid basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         invalidKey,
			basicAuth:   true,
			status:      http.StatusBadRequest,
		},
		"publish message without content type": {
			chanID:      chanID,
			msg:         msg,
			contentType: "",
			key:         thingKey,
			status:      http.StatusUnsupportedMediaType,
		},
		"publish message to invalid channel": {
			chanID:      "",
			msg:         msg,
			contentType: ctSenmlJSON,
			key:         thingKey,
			status:      http.StatusBadRequest,
		},
	}

	for desc, tc := range cases {
		t.Run(desc, func(t *testing.T) {
			req := testRequest{
				client:      ts.Client(),
				method:      http.MethodPost,
				url:         fmt.Sprintf("%s/channels/%s/messages", ts.URL, tc.chanID),
				contentType: tc.contentType,
				token:       tc.key,
				body:        strings.NewReader(tc.msg),
				basicAuth:   tc.basicAuth,
			}
			res, err := req.make()
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", desc, err))
			assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", desc, tc.status, res.StatusCode))
		})
	}
}
