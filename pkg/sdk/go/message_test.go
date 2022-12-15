// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/http/api"
	"github.com/mainflux/mainflux/http/mocks"
	"github.com/mainflux/mainflux/internal/apiutil"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	sdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
)

const eof = "EOF"

func newMessageService(cc mainflux.ThingsServiceClient) adapter.Service {
	pub := mocks.NewPublisher()
	return adapter.New(pub, cc)
}

func newMessageServer(svc adapter.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(svc, mocktracer.New(), logger)
	return httptest.NewServer(mux)
}

func TestSendMessage(t *testing.T) {
	chanID := "1"
	atoken := "auth_token"
	invalidToken := "invalid_token"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsClient(map[string]string{atoken: chanID})
	pub := newMessageService(thingsClient)
	ts := newMessageServer(pub)
	defer ts.Close()
	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := map[string]struct {
		chanID string
		msg    string
		auth   string
		err    errors.SDKError
	}{
		"publish message": {
			chanID: chanID,
			msg:    msg,
			auth:   atoken,
			err:    nil,
		},
		"publish message without authorization token": {
			chanID: chanID,
			msg:    msg,
			auth:   "",
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		"publish message with invalid authorization token": {
			chanID: chanID,
			msg:    msg,
			auth:   invalidToken,
			err:    errors.NewSDKErrorWithStatus(errors.New(eof), http.StatusUnauthorized),
		},
		"publish message with wrong content type": {
			chanID: chanID,
			msg:    "text",
			auth:   atoken,
			err:    nil,
		},
		"publish message to wrong channel": {
			chanID: "",
			msg:    msg,
			auth:   atoken,
			err:    errors.NewSDKErrorWithStatus(errors.ErrMalformedEntity, http.StatusBadRequest),
		},
		"publish message unable to authorize": {
			chanID: chanID,
			msg:    msg,
			auth:   "invalid-token",
			err:    errors.NewSDKErrorWithStatus(errors.New(eof), http.StatusUnauthorized),
		},
	}
	for desc, tc := range cases {
		err := mainfluxSDK.SendMessage(tc.chanID, tc.msg, tc.auth)
		if tc.err == nil {
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s", desc, err))
		} else {
			assert.Equal(t, tc.err.Error(), err.Error(), fmt.Sprintf("%s: expected error %s, got %s", desc, err, tc.err))
		}
	}
}

func TestSetContentType(t *testing.T) {
	chanID := "1"
	atoken := "auth_token"
	thingsClient := mocks.NewThingsClient(map[string]string{atoken: chanID})

	pub := newMessageService(thingsClient)
	ts := newMessageServer(pub)
	defer ts.Close()

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc  string
		cType sdk.ContentType
		err   errors.SDKError
	}{
		{
			desc:  "set senml+json content type",
			cType: "application/senml+json",
			err:   nil,
		},
		{
			desc:  "set invalid content type",
			cType: "invalid",
			err:   errors.NewSDKError(errors.ErrUnsupportedContentType),
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.SetContentType(tc.cType)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
