// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/absmach/magistrala/pkg/sdk"
	readersapi "github.com/absmach/magistrala/readers/api"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	chmocks "github.com/absmach/supermq/channels/mocks"
	climocks "github.com/absmach/supermq/clients/mocks"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	authnmocks "github.com/absmach/supermq/pkg/authn/mocks"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/transformers/senml"
	"github.com/absmach/supermq/readers"
	readersmocks "github.com/absmach/supermq/readers/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	channelsGRPCClient *chmocks.ChannelsServiceClient
	clientsGRPCClient  *climocks.ClientsServiceClient
)

func setupReaders() (*httptest.Server, *authnmocks.Authentication, *readersmocks.MessageRepository) {
	repo := new(readersmocks.MessageRepository)
	authn := new(authnmocks.Authentication)
	clientsGRPCClient = new(climocks.ClientsServiceClient)
	channelsGRPCClient = new(chmocks.ChannelsServiceClient)

	mux := readersapi.MakeHandler(repo, authn, clientsGRPCClient, channelsGRPCClient, "test", "")
	return httptest.NewServer(mux), authn, repo
}

func TestReadMessages(t *testing.T) {
	ts, authn, repo := setupReaders()
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
			authCall1 := authn.On("Authenticate", mock.Anything, tc.token).Return(smqauthn.Session{UserID: validID}, tc.authnErr)
			authzCall := channelsGRPCClient.On("Authorize", mock.Anything, mock.Anything).Return(&grpcChannelsV1.AuthzRes{Authorized: true}, tc.authzErr)
			repoCall := repo.On("ReadAll", channelID, mock.Anything).Return(tc.repoRes, tc.repoErr)
			response, err := mgsdk.ReadMessages(tc.messagePageMeta, tc.chanName, tc.domainID, tc.token)
			fmt.Println(err)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, response)
			if tc.err == nil {
				ok := repoCall.Parent.AssertCalled(t, "ReadAll", channelID, mock.Anything)
				assert.True(t, ok)
			}
			authCall1.Unset()
			authzCall.Unset()
			repoCall.Unset()
		})
	}
}
