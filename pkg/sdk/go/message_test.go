// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/magistrala"
	adapter "github.com/absmach/magistrala/http"
	"github.com/absmach/magistrala/http/api"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	authmocks "github.com/absmach/magistrala/pkg/auth/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	pubsub "github.com/absmach/magistrala/pkg/messaging/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	readersapi "github.com/absmach/magistrala/readers/api"
	readersmocks "github.com/absmach/magistrala/readers/mocks"
	thmocks "github.com/absmach/magistrala/things/mocks"
	"github.com/absmach/mproxy"
	mproxyhttp "github.com/absmach/mproxy/pkg/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupMessages() (*httptest.Server, *authmocks.AuthClient, *pubsub.PubSub) {
	auth := new(authmocks.AuthClient)
	pub := new(pubsub.PubSub)
	handler := adapter.NewHandler(pub, mglog.NewMock(), auth)

	mux := api.MakeHandler(mglog.NewMock(), "")
	target := httptest.NewServer(mux)

	config := mproxy.Config{
		Address: "",
		Target:  target.URL,
	}
	mp, err := mproxyhttp.NewProxy(config, handler, mglog.NewMock())
	if err != nil {
		return nil, nil, nil
	}

	return httptest.NewServer(http.HandlerFunc(mp.ServeHTTP)), auth, pub
}

func setupReader() (*httptest.Server, *authmocks.AuthClient, *readersmocks.MessageRepository) {
	repo := new(readersmocks.MessageRepository)
	auth := new(authmocks.AuthClient)
	tauth := new(thmocks.AuthzServiceClient)

	mux := readersapi.MakeHandler(repo, auth, tauth, "test", "")
	return httptest.NewServer(mux), auth, repo
}

func TestSendMessage(t *testing.T) {
	ts, auth, pub := setupMessages()
	defer ts.Close()

	msg := `[{"n":"current","t":-1,"v":1.6}]`
	thingKey := "thingKey"
	channelID := "channelID"

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		chanName string
		msg      string
		thingKey string
		authRes  *magistrala.AuthorizeRes
		authErr  error
		svcErr   error
		err      errors.SDKError
	}{
		{
			desc:     "publish message successfully",
			chanName: channelID,
			msg:      msg,
			thingKey: thingKey,
			authRes:  &magistrala.AuthorizeRes{Authorized: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "publish message with empty thing key",
			chanName: channelID,
			msg:      msg,
			thingKey: "",
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			authErr:  svcerr.ErrAuthorization,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:     "publish message with invalid thing key",
			chanName: channelID,
			msg:      msg,
			thingKey: "invalid",
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			authErr:  svcerr.ErrAuthorization,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:     "publish message with invalid channel ID",
			chanName: wrongID,
			msg:      msg,
			thingKey: thingKey,
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			authErr:  svcerr.ErrAuthorization,
			svcErr:   svcerr.ErrAuthorization,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:     "publish message with empty message body",
			chanName: channelID,
			msg:      "",
			thingKey: thingKey,
			authRes:  &magistrala.AuthorizeRes{Authorized: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyMessage), http.StatusBadRequest),
		},
		{
			desc:     "publish message with channel subtopic",
			chanName: channelID + ".subtopic",
			msg:      msg,
			thingKey: thingKey,
			authRes:  &magistrala.AuthorizeRes{Authorized: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authRes, tc.authErr)
			svcCall := pub.On("Publish", mock.Anything, channelID, mock.Anything).Return(tc.svcErr)
			err := mgsdk.SendMessage(tc.chanName, tc.msg, tc.thingKey)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Publish", mock.Anything, channelID, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestSetContentType(t *testing.T) {
	ts, _, _ := setupMessages()
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

func TestReadMessages(t *testing.T) {
	ts, auth, repo := setupReader()
	defer ts.Close()

	channelID := "channelID"
	msgValue := 1.6
	boolVal := true
	msg := senml.Message{
		Name:      "current",
		Time:      1720000000,
		Value:     &msgValue,
		Publisher: validID,
	}
	invalidMsg := "[{\"n\":\"current\",\"t\":-1,\"v\":1.6}]"

	sdkConf := sdk.Config{
		ReaderURL: ts.URL,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc            string
		token           string
		chanName        string
		messagePageMeta sdk.MessagePageMetadata
		authRes         *magistrala.AuthorizeRes
		authErr         error
		repoRes         readers.MessagesPage
		repoErr         error
		response        sdk.MessagesPage
		err             errors.SDKError
	}{
		{
			desc:     "read messages successfully",
			token:    validToken,
			chanName: channelID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
					Level:  0,
				},
				Publisher: validID,
				BoolValue: &boolVal,
			},
			authRes: &magistrala.AuthorizeRes{Authorized: true, Id: validID},
			repoRes: readers.MessagesPage{
				Total:    1,
				Messages: []readers.Message{msg},
			},
			repoErr: nil,
			response: sdk.MessagesPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Messages: []senml.Message{msg},
			},
			err: nil,
		},
		{
			desc:     "read messages successfully with subtopic",
			token:    validToken,
			chanName: channelID + ".subtopic",
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Publisher: validID,
			},
			authRes: &magistrala.AuthorizeRes{Authorized: true, Id: validID},
			repoRes: readers.MessagesPage{
				Total:    1,
				Messages: []readers.Message{msg},
			},
			repoErr: nil,
			response: sdk.MessagesPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Messages: []senml.Message{msg},
			},
			err: nil,
		},
		{
			desc:     "read messages with invalid token",
			token:    invalidToken,
			chanName: channelID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			authErr:  svcerr.ErrAuthorization,
			repoRes:  readers.MessagesPage{},
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrAuthorization, svcerr.ErrAuthorization), http.StatusUnauthorized),
		},
		{
			desc:     "read messages with empty token",
			token:    "",
			chanName: channelID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			authErr:  svcerr.ErrAuthorization,
			repoRes:  readers.MessagesPage{},
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "read messages with empty channel ID",
			token:    validToken,
			chanName: "",
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			repoRes:  readers.MessagesPage{},
			repoErr:  nil,
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "read messages with invalid message page metadata",
			token:    validToken,
			chanName: channelID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
					Metadata: map[string]interface{}{
						"key": make(chan int),
					},
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authRes:  &magistrala.AuthorizeRes{Authorized: false, Id: ""},
			repoRes:  readers.MessagesPage{},
			repoErr:  nil,
			response: sdk.MessagesPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "read messages with response that cannot be unmarshalled",
			token:    validToken,
			chanName: channelID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authRes: &magistrala.AuthorizeRes{Authorized: true, Id: validID},
			repoRes: readers.MessagesPage{
				Total:    1,
				Messages: []readers.Message{invalidMsg},
			},
			repoErr:  nil,
			response: sdk.MessagesPage{},
			err:      errors.NewSDKError(errors.New("json: cannot unmarshal string into Go struct field MessagesPage.messages of type senml.Message")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(tc.authRes, tc.authErr)
			repoCall := repo.On("ReadAll", channelID, mock.Anything).Return(tc.repoRes, tc.repoErr)
			response, err := mgsdk.ReadMessages(tc.messagePageMeta, tc.chanName, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, response)
			if tc.err == nil {
				ok := repoCall.Parent.AssertCalled(t, "ReadAll", channelID, mock.Anything)
				assert.True(t, ok)
			}
			authCall.Unset()
			repoCall.Unset()
		})
	}
}
