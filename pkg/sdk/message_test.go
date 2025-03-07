// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/absmach/mgate"
	proxy "github.com/absmach/mgate/pkg/http"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	climocks "github.com/absmach/supermq/clients/mocks"
	adapter "github.com/absmach/supermq/http"
	"github.com/absmach/supermq/http/api"
	smqlog "github.com/absmach/supermq/logger"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	pubsub "github.com/absmach/supermq/pkg/messaging/mocks"
	sdk "github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	channelsGRPCClient *chmocks.ChannelsServiceClient
	clientsGRPCClient  *climocks.ClientsServiceClient
)

func setupMessages() (*httptest.Server, *pubsub.PubSub) {
	clientsGRPCClient = new(climocks.ClientsServiceClient)
	channelsGRPCClient = new(chmocks.ChannelsServiceClient)
	pub := new(pubsub.PubSub)
	authn := new(authnmocks.Authentication)
	handler := adapter.NewHandler(pub, authn, clientsGRPCClient, channelsGRPCClient, smqlog.NewMock())

	mux := api.MakeHandler(smqlog.NewMock(), "")
	target := httptest.NewServer(mux)

	config := mgate.Config{
		Address: "",
		Target:  target.URL,
	}
	mp, err := proxy.NewProxy(config, handler, smqlog.NewMock())
	if err != nil {
		return nil, nil
	}

	return httptest.NewServer(http.HandlerFunc(mp.ServeHTTP)), pub
}

func TestSendMessage(t *testing.T) {
	ts, pub := setupMessages()
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
			authErr:   svcerr.ErrAuthentication,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "publish message with invalid client key",
			chanName:  channelID,
			msg:       msg,
			clientKey: "invalid",
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			authErr:   svcerr.ErrAuthentication,
			svcErr:    svcerr.ErrAuthentication,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "publish message with invalid channel ID",
			chanName:  wrongID,
			msg:       msg,
			clientKey: clientKey,
			authRes:   &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			authErr:   svcerr.ErrAuthentication,
			svcErr:    svcerr.ErrAuthentication,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
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
			authzCall := clientsGRPCClient.On("Authenticate", mock.Anything, mock.Anything).Return(tc.authRes, tc.authErr)
			authnCall := channelsGRPCClient.On("Authorize", mock.Anything, mock.Anything).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)
			svcCall := pub.On("Publish", mock.Anything, channelID, mock.Anything).Return(tc.svcErr)
			err := mgsdk.SendMessage(tc.chanName, tc.msg, tc.clientKey)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Publish", mock.Anything, channelID, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authzCall.Unset()
			authnCall.Unset()
		})
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
