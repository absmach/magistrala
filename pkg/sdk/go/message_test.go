// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	chmocks "github.com/absmach/magistrala/channels/mocks"
	climocks "github.com/absmach/magistrala/clients/mocks"
	adapter "github.com/absmach/magistrala/http"
	"github.com/absmach/magistrala/http/api"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	pubsub "github.com/absmach/magistrala/pkg/messaging/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	readersapi "github.com/absmach/magistrala/readers/api"
	readersmocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/absmach/mgate"
	proxy "github.com/absmach/mgate/pkg/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupMessages() (*httptest.Server, *climocks.ClientsServiceClient, *pubsub.PubSub) {
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)
	pub := new(pubsub.PubSub)
	authn := new(authnmocks.Authentication)
	handler := adapter.NewHandler(pub, authn, clients, channels, mglog.NewMock())

	mux := api.MakeHandler(mglog.NewMock(), "")
	target := httptest.NewServer(mux)

	config := mgate.Config{
		Address: "",
		Target:  target.URL,
	}
	mp, err := proxy.NewProxy(config, handler, mglog.NewMock())
	if err != nil {
		return nil, nil, nil
	}

	return httptest.NewServer(http.HandlerFunc(mp.ServeHTTP)), clients, pub
}

func setupReader() (*httptest.Server, *authnmocks.Authentication, *readersmocks.MessageRepository) {
	repo := new(readersmocks.MessageRepository)
	authn := new(authnmocks.Authentication)
	clients := new(climocks.ClientsServiceClient)
	channels := new(chmocks.ChannelsServiceClient)

	mux := readersapi.MakeHandler(repo, authn, clients, channels, "test", "")
	return httptest.NewServer(mux), authn, repo
}

func TestSendMessage(t *testing.T) {
	ts, clients, pub := setupMessages()
	defer ts.Close()

	msg := `[{"n":"current","t":-1,"v":1.6}]`
	clientKey := "clientKey"
	channelID := "channelID"

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc      string
		chanName  string
		msg       string
		clientKey string
		authRes   *grpcClientsV1.AuthnRes
		authErr   error
		svcErr    error
		err       errors.SDKError
	}{
		{
			desc:      "publish message successfully",
			chanName:  channelID,
			msg:       msg,
			clientKey: clientKey,
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:   nil,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:      "publish message with empty client key",
			chanName:  channelID,
			msg:       msg,
			clientKey: "",
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			authErr:   svcerr.ErrAuthorization,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:      "publish message with invalid client key",
			chanName:  channelID,
			msg:       msg,
			clientKey: "invalid",
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			authErr:   svcerr.ErrAuthorization,
			svcErr:    svcerr.ErrAuthorization,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:      "publish message with invalid channel ID",
			chanName:  wrongID,
			msg:       msg,
			clientKey: clientKey,
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			authErr:   svcerr.ErrAuthorization,
			svcErr:    svcerr.ErrAuthorization,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusBadRequest),
		},
		{
			desc:      "publish message with empty message body",
			chanName:  channelID,
			msg:       "",
			clientKey: clientKey,
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:   nil,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyMessage), http.StatusBadRequest),
		},
		{
			desc:      "publish message with channel subtopic",
			chanName:  channelID + ".subtopic",
			msg:       msg,
			clientKey: clientKey,
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:   nil,
			svcErr:    nil,
			err:       nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			authCall := clients.On("Authorize", mock.Anything, mock.Anything).Return(tc.authRes, tc.authErr)
			svcCall := pub.On("Publish", mock.Anything, channelID, mock.Anything).Return(tc.svcErr)
			err := mgsdk.SendMessage(tc.chanName, tc.msg, tc.clientKey)
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
	ts, authz, authn, repo := setupReader()
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
		domainID        string
		messagePageMeta sdk.MessagePageMetadata
		authzErr        error
		authnErr        error
		repoRes         readers.MessagesPage
		repoErr         error
		response        sdk.MessagesPage
		err             errors.SDKError
	}{
		{
			desc:     "read messages successfully",
			token:    validToken,
			chanName: channelID,
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
					Level:  0,
				},
				Publisher: validID,
				BoolValue: &boolVal,
			},
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
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Publisher: validID,
			},
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
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authzErr: svcerr.ErrAuthorization,
			repoRes:  readers.MessagesPage{},
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(svcerr.ErrAuthorization, svcerr.ErrAuthorization), http.StatusUnauthorized),
		},
		{
			desc:     "read messages with empty token",
			token:    "",
			chanName: channelID,
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			authnErr: svcerr.ErrAuthentication,
			repoRes:  readers.MessagesPage{},
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
		},
		{
			desc:     "read messages with empty channel ID",
			token:    validToken,
			chanName: "",
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
			repoRes:  readers.MessagesPage{},
			repoErr:  nil,
			response: sdk.MessagesPage{},
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "read messages with invalid message page metadata",
			token:    validToken,
			chanName: channelID,
			domainID: validID,
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
			repoRes:  readers.MessagesPage{},
			repoErr:  nil,
			response: sdk.MessagesPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "read messages with response that cannot be unmarshalled",
			token:    validToken,
			chanName: channelID,
			domainID: validID,
			messagePageMeta: sdk.MessagePageMetadata{
				PageMetadata: sdk.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
				Subtopic:  "subtopic",
				Publisher: validID,
			},
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
			authCall := authz.On("Authorize", mock.Anything, mock.Anything).Return(tc.authzErr)
			authCall1 := authn.On("Authenticate", mock.Anything, tc.token).Return(mgauthn.Session{UserID: validID}, tc.authnErr)
			repoCall := repo.On("ReadAll", channelID, mock.Anything).Return(tc.repoRes, tc.repoErr)
			response, err := mgsdk.ReadMessages(tc.messagePageMeta, tc.chanName, tc.domainID, tc.token)
			fmt.Println(err)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, response)
			if tc.err == nil {
				ok := repoCall.Parent.AssertCalled(t, "ReadAll", channelID, mock.Anything)
				assert.True(t, ok)
			}
			authCall.Unset()
			authCall1.Unset()
			repoCall.Unset()
		})
	}
}
