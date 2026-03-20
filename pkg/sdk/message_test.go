// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/absmach/mgate"
	proxy "github.com/absmach/mgate/pkg/http"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	grpcCommonV1 "github.com/absmach/supermq/api/grpc/common/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	climocks "github.com/absmach/supermq/clients/mocks"
	dmocks "github.com/absmach/supermq/domains/mocks"
	adapter "github.com/absmach/supermq/http"
	"github.com/absmach/supermq/http/api"
	httpmocks "github.com/absmach/supermq/http/mocks"
	smqlog "github.com/absmach/supermq/logger"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	pubsub "github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	channelsGRPCClient *chmocks.ChannelsServiceClient
	clientsGRPCClient  *climocks.ClientsServiceClient
	domainsGRPCClient  *dmocks.DomainsServiceClient
)

func setupMessages(t *testing.T) (*httptest.Server, *pubsub.PubSub) {
	clientsGRPCClient = new(climocks.ClientsServiceClient)
	channelsGRPCClient = new(chmocks.ChannelsServiceClient)
	domainsGRPCClient = new(dmocks.DomainsServiceClient)
	pub := new(pubsub.PubSub)
	authn := new(authnmocks.Authentication)
	svc := new(httpmocks.Service)

	parser, err := messaging.NewTopicParser(messaging.DefaultCacheConfig, channelsGRPCClient, domainsGRPCClient)
	assert.Nil(t, err, fmt.Sprintf("unexpected error while setting up parser: %v", err))
	handler := adapter.NewHandler(pub, smqlog.NewMock(), authn, clientsGRPCClient, channelsGRPCClient, parser)
	resolver := messaging.NewTopicResolver(channelsGRPCClient, domainsGRPCClient)

	mux := api.MakeHandler(context.Background(), svc, resolver, smqlog.NewMock(), "")
	target := httptest.NewServer(mux)

	ptUrl, _ := url.Parse(target.URL)
	ptHost, ptPort, _ := net.SplitHostPort(ptUrl.Host)
	config := mgate.Config{
		Host:           "",
		Port:           "",
		PathPrefix:     "",
		TargetHost:     ptHost,
		TargetPort:     ptPort,
		TargetProtocol: ptUrl.Scheme,
		TargetPath:     ptUrl.Path,
	}

	mp, err := proxy.NewProxy(config, handler, smqlog.NewMock(), []string{}, []string{"/health", "/metrics"})
	if err != nil {
		return nil, nil
	}

	return httptest.NewServer(http.HandlerFunc(mp.ServeHTTP)), pub
}

func TestSendMessage(t *testing.T) {
	ts, pub := setupMessages(t)
	defer ts.Close()

	msg := `[{"n":"current","t":-1,"v":1.6}]`
	clientKey := "clientKey"
	channelID := "channelID"
	domainID := "domainID"

	sdkConf := sdk.Config{
		HTTPAdapterURL:  ts.URL,
		MsgContentType:  "application/senml+json",
		TLSVerification: false,
	}

	mgsdk := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc     string
		topic    string
		domainID string
		msg      string
		secret   string
		authRes  *grpcClientsV1.AuthnRes
		authErr  error
		svcErr   error
		err      errors.SDKError
	}{
		{
			desc:     "publish message successfully",
			topic:    channelID,
			domainID: domainID,
			msg:      msg,
			secret:   clientKey,
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "publish message with empty client key",
			topic:    channelID,
			domainID: domainID,
			msg:      msg,
			secret:   "",
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "publish message with invalid client key",
			topic:    channelID,
			domainID: domainID,
			msg:      msg,
			secret:   "invalid",
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "publish message with invalid channel ID",
			topic:    wrongID,
			domainID: domainID,
			msg:      msg,
			secret:   clientKey,
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "publish message with empty message body",
			topic:    channelID,
			domainID: domainID,
			msg:      "",
			secret:   clientKey,
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrEmptyMessage, http.StatusBadRequest),
		},
		{
			desc:     "publish message with channel subtopic",
			topic:    channelID + ".subtopic",
			domainID: domainID,
			msg:      msg,
			secret:   clientKey,
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: true, Id: ""},
			authErr:  nil,
			svcErr:   nil,
			err:      nil,
		},
		{
			desc:     "publish message with invalid domain ID",
			topic:    channelID,
			domainID: wrongID,
			msg:      msg,
			secret:   clientKey,
			authRes:  &grpcClientsV1.AuthnRes{Authenticated: false, Id: ""},
			svcErr:   svcerr.ErrAuthentication,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
	}
	for _, tc := range cases {
		internalTopic := tc.domainID + ".c." + strings.ReplaceAll(tc.topic, "/", ".")
		t.Run(tc.desc, func(t *testing.T) {
			authzCall := clientsGRPCClient.On("Authenticate", mock.Anything, mock.Anything).Return(tc.authRes, tc.authErr)
			authnCall := channelsGRPCClient.On("Authorize", mock.Anything, mock.Anything).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, nil)
			svcCall := pub.On("Publish", mock.Anything, internalTopic, mock.Anything).Return(tc.svcErr)
			domainsCall := domainsGRPCClient.On("RetrieveIDByRoute", mock.Anything, mock.Anything).Return(&grpcCommonV1.RetrieveEntityRes{Entity: &grpcCommonV1.EntityBasic{Id: tc.domainID}}, nil)
			channelsCall := channelsGRPCClient.On("RetrieveIDByRoute", mock.Anything, mock.Anything).Return(&grpcCommonV1.RetrieveEntityRes{Entity: &grpcCommonV1.EntityBasic{Id: channelID}}, nil)
			err := mgsdk.SendMessage(context.Background(), tc.domainID, tc.topic, tc.msg, tc.secret)
			if tc.err != nil {
				assert.Contains(t, err.Error(), tc.err.Error(), fmt.Sprintf("expected error message to contain: %v, got: %v", tc.err, err))
			}
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Publish", mock.Anything, internalTopic, mock.Anything)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authzCall.Unset()
			authnCall.Unset()
			domainsCall.Unset()
			channelsCall.Unset()
		})
	}
}

func TestSetContentType(t *testing.T) {
	ts, _ := setupMessages(t)
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
