// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	adapter "github.com/absmach/magistrala/http"
	"github.com/absmach/magistrala/http/api"
	"github.com/absmach/magistrala/http/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	mproxy "github.com/absmach/mproxy/pkg/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupMessages() (*httptest.Server, *authmocks.Service) {
	auth := new(authmocks.Service)
	pub := mocks.NewPublisher()
	handler := adapter.NewHandler(pub, mglog.NewMock(), auth)

	mux := api.MakeHandler("")
	target := httptest.NewServer(mux)

	mp, err := mproxy.NewProxy("", target.URL, handler, mglog.NewMock())
	if err != nil {
		return nil, nil
	}

	return httptest.NewServer(http.HandlerFunc(mp.Handler)), auth
}

func TestSendMessage(t *testing.T) {
	chanID := "1"
	atoken := "auth_token"
	invalidToken := "invalid_token"
	msg := `[{"n":"current","t":-1,"v":1.6}]`

	ts, auth := setupMessages()
	defer ts.Close()
	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{Subject: atoken, Object: chanID, Domain: "", SubjectType: "thing", Permission: "publish", ObjectType: "group"}).Return(&magistrala.AuthorizeRes{Authorized: true, Id: ""}, nil)
	auth.On("Authorize", mock.Anything, &magistrala.AuthorizeReq{Subject: invalidToken, Object: chanID, Domain: "", SubjectType: "thing", Permission: "publish", ObjectType: "group"}).Return(&magistrala.AuthorizeRes{Authorized: true, Id: ""}, errors.ErrAuthentication)
	auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false, Id: ""}, nil)

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
			err:    errors.NewSDKErrorWithStatus(errors.ErrAuthorization, http.StatusBadRequest),
		},
		"publish message with invalid authorization token": {
			chanID: chanID,
			msg:    msg,
			auth:   invalidToken,
			err:    errors.NewSDKErrorWithStatus(errors.ErrAuthentication, http.StatusBadRequest),
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
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(adapter.ErrFailedPublish, adapter.ErrMalformedTopic), http.StatusBadRequest),
		},
		"publish message unable to authorize": {
			chanID: chanID,
			msg:    msg,
			auth:   "invalid-token",
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
	}
	for desc, tc := range cases {
		err := mgsdk.SendMessage(tc.chanID, tc.msg, tc.auth)
		switch tc.err {
		case nil:
			assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s", desc, err))
		default:
			assert.Equal(t, tc.err.Error(), err.Error(), fmt.Sprintf("%s: expected error %s, got %s", desc, tc.err, err))
		}
	}
}

func TestSetContentType(t *testing.T) {
	ts, _ := setupMessages()
	defer ts.Close()

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}
	mgsdk := sdk.NewSDK(sdkConf)

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
			err:   errors.NewSDKError(apiutil.ErrUnsupportedContentType),
		},
	}
	for _, tc := range cases {
		err := mgsdk.SetContentType(tc.cType)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
