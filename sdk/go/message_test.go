//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package sdk_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/mainflux/mainflux"
	adapter "github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/http/api"
	"github.com/mainflux/mainflux/http/mocks"
	sdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/stretchr/testify/assert"
)

func newMessageService() mainflux.MessagePublisher {
	pub := mocks.NewPublisher()
	return adapter.New(pub)
}

func newMessageServer(pub mainflux.MessagePublisher, cc mainflux.ThingsServiceClient) *httptest.Server {
	mux := api.MakeHandler(pub, cc)
	return httptest.NewServer(mux)
}

func TestSendMessage(t *testing.T) {
	chanID := "1"
	atoken := "auth_token"
	invalidToken := "invalid_token"
	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingsClient := mocks.NewThingsClient(map[string]string{atoken: chanID})
	pub := newMessageService()
	ts := newMessageServer(pub, thingsClient)
	defer ts.Close()
	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := map[string]struct {
		chanID string
		msg    string
		auth   string
		err    error
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
			err:    sdk.ErrUnauthorized,
		},
		"publish message with invalid authorization token": {
			chanID: chanID,
			msg:    msg,
			auth:   invalidToken,
			err:    sdk.ErrUnauthorized,
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
			err:    sdk.ErrInvalidArgs,
		},
		"publish message unable to authorize": {
			chanID: chanID,
			msg:    msg,
			auth:   mocks.ServiceErrToken,
			err:    sdk.ErrFailedPublish,
		},
	}
	for desc, tc := range cases {
		err := mainfluxSDK.SendMessage(tc.chanID, tc.msg, tc.auth)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
	}
}

func TestSetContentType(t *testing.T) {
	chanID := "1"
	atoken := "auth_token"
	thingsClient := mocks.NewThingsClient(map[string]string{atoken: chanID})

	pub := newMessageService()
	ts := newMessageServer(pub, thingsClient)
	defer ts.Close()

	sdkConf := sdk.Config{
		BaseURL:           ts.URL,
		UsersPrefix:       "",
		ThingsPrefix:      "",
		HTTPAdapterPrefix: "",
		MsgContentType:    contentType,
		TLSVerification:   false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc  string
		cType sdk.ContentType
		err   error
	}{
		{
			desc:  "set senml+json content type",
			cType: "application/senml+json",
			err:   nil,
		},
		{
			desc:  "set invalid content type",
			cType: "invalid",
			err:   sdk.ErrInvalidContentType,
		},
	}
	for _, tc := range cases {
		err := mainfluxSDK.SetContentType(tc.cType)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
