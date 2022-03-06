// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/http/api"
	"github.com/mainflux/mainflux/http/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
)

func newService(cc mainflux.ThingsServiceClient) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, cc)
}

func newHTTPServer(svc adapter.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
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
	chanID := "1"
	contentType := "application/senml+json"
	thingKey := "thing_key"
	invalidKey := "invalid_key"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsClient(map[string]string{thingKey: chanID})
	svc := newService(thingsClient)
	ts := newHTTPServer(svc)
	defer ts.Close()

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
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusAccepted,
		},
		"publish message with empty key": {
			chanID:      chanID,
			msg:         msg,
			contentType: contentType,
			key:         "",
			status:      http.StatusUnauthorized,
		},
		"publish message with basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: contentType,
			key:         thingKey,
			basicAuth:   true,
			status:      http.StatusAccepted,
		},
		"publish message with invalid key": {
			chanID:      chanID,
			msg:         msg,
			contentType: contentType,
			key:         invalidKey,
			status:      http.StatusUnauthorized,
		},
		"publish message with invalid basic auth": {
			chanID:      chanID,
			msg:         msg,
			contentType: contentType,
			key:         invalidKey,
			basicAuth:   true,
			status:      http.StatusUnauthorized,
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
			contentType: contentType,
			key:         thingKey,
			status:      http.StatusBadRequest,
		},
		"publish message unable to authorize": {
			chanID:      chanID,
			msg:         msg,
			contentType: contentType,
			key:         mocks.ServiceErrToken,
			status:      http.StatusInternalServerError,
		},
	}

	for desc, tc := range cases {
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
	}
}
