// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/channels"
	chmocks "github.com/absmach/magistrala/channels/mocks"
	"github.com/absmach/magistrala/clients"
	thapi "github.com/absmach/magistrala/clients/api/http"
	climocks "github.com/absmach/magistrala/clients/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	authnmocks "github.com/absmach/magistrala/pkg/authn/mocks"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	oauth2mocks "github.com/absmach/magistrala/pkg/oauth2/mocks"
	policies "github.com/absmach/magistrala/pkg/policies"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	usapi "github.com/absmach/magistrala/users/api"
	usmocks "github.com/absmach/magistrala/users/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	channelName    = "channelName"
	newName        = "newName"
	newDescription = "newDescription"
	channel        = generateTestChannel(&testing.T{})
)

func setupChannels() (*httptest.Server, *chmocks.Service, *authnmocks.Authentication) {
	tsvc := new(climocks.Service)
	usvc := new(usmocks.Service)
	gsvc := new(chmocks.Service)
	logger := mglog.NewMock()
	provider := new(oauth2mocks.Provider)
	provider.On("Name").Return("test")
	authn := new(authnmocks.Authentication)
	token := new(authmocks.TokenServiceClient)

	mux := chi.NewRouter()

	thapi.MakeHandler(tsvc, authn, mux, logger, "")
	usapi.MakeHandler(usvc, authn, token, true, gsvc, mux, logger, "", passRegex, provider)
	return httptest.NewServer(mux), gsvc, authn
}

func TestCreateChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	channel := convertChannel(channel)
	createGroupReq := channels.Channel{
		Name:     channel.Name,
		Metadata: clients.Metadata{"role": "client"},
		Status:   clients.EnabledStatus,
	}

	channelReq := sdk.Channel{
		Name:     channel.Name,
		Metadata: validMetadata,
		Status:   clients.EnabledStatus.String(),
	}

	channelKind := "new_channel"
	parentID := testsutil.GenerateUUID(&testing.T{})
	pGroup := group
	pGroup.Parent = parentID
	pChannel := channel
	pChannel.ParentID = parentID

	iGroup := group
	iGroup.Metadata = clients.Metadata{
		"test": make(chan int),
	}

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	cases := []struct {
		desc            string
		channelReq      sdk.Channel
		domainID        string
		token           string
		session         mgauthn.Session
		createGroupReq  channels.Channel
		svcRes          channels.Channel
		svcErr          error
		authenticateRes mgauthn.Session
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:           "create channel successfully",
			channelReq:     channelReq,
			domainID:       domainID,
			token:          validToken,
			createGroupReq: createGroupReq,
			svcRes:         group,
			svcErr:         nil,
			response:       channel,
			err:            nil,
		},
		{
			desc:           "create channel with existing name",
			channelReq:     channelReq,
			domainID:       domainID,
			token:          validToken,
			createGroupReq: createGroupReq,
			svcRes:         channels.Channel{},
			svcErr:         svcerr.ErrCreateEntity,
			response:       sdk.Channel{},
			err:            errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "create channel that can't be marshalled",
			channelReq: sdk.Channel{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			domainID:       domainID,
			token:          validToken,
			createGroupReq: channels.Channel{},
			svcRes:         channels.Channel{},
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc: "create channel with parent",
			channelReq: sdk.Channel{
				Name:     channel.Name,
				ParentID: parentID,
				Status:   clients.EnabledStatus.String(),
			},
			domainID: domainID,
			token:    validToken,
			createGroupReq: channels.Channel{
				Name:   channel.Name,
				Parent: parentID,
				Status: clients.EnabledStatus,
			},
			svcRes:   pGroup,
			svcErr:   nil,
			response: pChannel,
			err:      nil,
		},
		{
			desc: "create channel with invalid parent",
			channelReq: sdk.Channel{
				Name:     channel.Name,
				ParentID: wrongID,
				Status:   clients.EnabledStatus.String(),
			},
			domainID: domainID,
			token:    validToken,
			createGroupReq: channels.Channel{
				Name:   channel.Name,
				Parent: wrongID,
				Status: clients.EnabledStatus,
			},
			svcRes:   channels.Channel{},
			svcErr:   svcerr.ErrCreateEntity,
			response: sdk.Channel{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "create channel with missing name",
			channelReq: sdk.Channel{
				Status: clients.EnabledStatus.String(),
			},
			domainID:       domainID,
			token:          validToken,
			createGroupReq: channels.Channel{},
			svcRes:         channels.Channel{},
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc: "create a channel with every field defined",
			channelReq: sdk.Channel{
				ID:          group.ID,
				ParentID:    parentID,
				Name:        channel.Name,
				Description: description,
				Metadata:    validMetadata,
				CreatedAt:   group.CreatedAt,
				UpdatedAt:   group.UpdatedAt,
				Status:      clients.EnabledStatus.String(),
			},
			domainID: domainID,
			token:    validToken,
			createGroupReq: channels.Channel{
				ID:          group.ID,
				Parent:      parentID,
				Name:        channel.Name,
				Description: description,
				Metadata:    clients.Metadata{"role": "client"},
				CreatedAt:   group.CreatedAt,
				UpdatedAt:   group.UpdatedAt,
				Status:      clients.EnabledStatus,
			},
			svcRes:   pGroup,
			svcErr:   nil,
			response: pChannel,
			err:      nil,
		},
		{
			desc:           "create channel with response that can't be unmarshalled",
			channelReq:     channelReq,
			domainID:       domainID,
			token:          validToken,
			createGroupReq: createGroupReq,
			svcRes:         iGroup,
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("CreateGroup", mock.Anything, tc.session, channelKind, tc.createGroupReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.CreateChannel(tc.channelReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "CreateGroup", mock.Anything, tc.session, channelKind, tc.createGroupReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListChannels(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	var chs []sdk.Channel
	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		gr := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("channel_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("client_%d", i)},
			Status:   clients.EnabledStatus.String(),
		}
		chs = append(chs, gr)
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		status          channels.Status
		total           uint64
		offset          uint64
		limit           uint64
		level           int
		name            string
		metadata        sdk.Metadata
		groupsPageMeta  channels.Page
		svcRes          channels.Page
		svcErr          error
		authenticateRes mgauthn.Session
		authenticateErr error
		response        sdk.ChannelsPage
		err             errors.SDKError
	}{
		{
			desc:     "list channels successfully",
			token:    validToken,
			domainID: domainID,
			limit:    limit,
			offset:   offset,
			total:    total,
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: offset,
					Limit:  limit,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: uint64(len(chs[offset:limit])),
				},
				Groups: convertChannels(chs[offset:limit]),
			},
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(chs[offset:limit])),
				},
				Channels: chs[offset:limit],
			},
			err: nil,
		},
		{
			desc:     "list channels with invalid token",
			token:    invalidToken,
			domainID: domainID,
			offset:   offset,
			limit:    limit,
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: offset,
					Limit:  limit,
				},
			},
			svcRes:          channels.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ChannelsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:           "list channels with empty token",
			token:          "",
			domainID:       validID,
			offset:         offset,
			limit:          limit,
			groupsPageMeta: channels.Page{},
			svcRes:         channels.Page{},
			svcErr:         nil,
			response:       sdk.ChannelsPage{},
			err:            errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list channels with zero limit",
			token:    validToken,
			domainID: domainID,
			offset:   offset,
			limit:    0,
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: offset,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: uint64(len(chs[offset:])),
				},
				Groups: convertChannels(chs[offset:limit]),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(chs[offset:])),
				},
				Channels: chs[offset:limit],
			},
			err: nil,
		},
		{
			desc:           "list channels with limit greater than max",
			token:          validToken,
			domainID:       domainID,
			offset:         offset,
			limit:          110,
			groupsPageMeta: channels.Page{},
			svcRes:         channels.Page{},
			svcErr:         nil,
			response:       sdk.ChannelsPage{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:     "list channels with level",
			token:    validToken,
			domainID: domainID,
			offset:   0,
			limit:    1,
			level:    1,
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: offset,
					Limit:  1,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Groups: convertChannels(chs[0:1]),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Channels: chs[0:1],
			},
			err: nil,
		},
		{
			desc:     "list channels with metadata",
			token:    validToken,
			domainID: domainID,
			offset:   0,
			limit:    10,
			metadata: sdk.Metadata{"name": "client_89"},
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset:   offset,
					Limit:    10,
					Metadata: clients.Metadata{"name": "client_89"},
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Groups: convertChannels([]sdk.Channel{chs[89]}),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Channels: []sdk.Channel{chs[89]},
			},
			err: nil,
		},
		{
			desc:     "list channels with invalid metadata",
			token:    validToken,
			domainID: domainID,
			offset:   0,
			limit:    10,
			metadata: sdk.Metadata{
				"test": make(chan int),
			},
			groupsPageMeta: channels.Page{},
			svcRes:         channels.Page{},
			svcErr:         nil,
			response:       sdk.ChannelsPage{},
			err:            errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "list channels with service response that can't be unmarshalled",
			token:    validToken,
			domainID: domainID,
			offset:   0,
			limit:    10,
			groupsPageMeta: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Groups: []channels.Channel{{
					ID: generateUUID(t),
					Metadata: clients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			pm := sdk.PageMetadata{
				Offset:   tc.offset,
				Limit:    tc.limit,
				Level:    uint64(tc.level),
				Metadata: tc.metadata,
			}
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.groupsPageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Channels(pm, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.UsersKind, "", tc.groupsPageMeta)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestViewChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	groupRes := convertChannel(channel)
	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		svcRes          channels.Channel
		svcErr          error
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:      "view channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: groupRes.ID,
			svcRes:    groupRes,
			svcErr:    nil,
			response:  channel,
			err:       nil,
		},
		{
			desc:            "view channel with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       groupRes.ID,
			svcRes:          channels.Channel{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Channel{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "view channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: groupRes.ID,
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "view channel for wrong id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrViewEntity,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:      "view channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKError(apiutil.ErrMissingID),
		},
		{
			desc:      "view channel with service response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			channelID: groupRes.ID,
			svcRes: channels.Channel{
				ID: generateUUID(t),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Channel{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ViewGroup", mock.Anything, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.Channel(tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroup", mock.Anything, tc.session, tc.channelID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestUpdateChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group := convertChannel(channel)
	nGroup := group
	nGroup.Name = newName
	nChannel := channel
	nChannel.Name = newName

	dGroup := group
	dGroup.Description = newDescription
	dChannel := channel
	dChannel.Description = newDescription

	mGroup := group
	mGroup.Metadata = clients.Metadata{
		"field": "value2",
	}
	mChannel := channel
	mChannel.Metadata = sdk.Metadata{
		"field": "value2",
	}

	aGroup := group
	aGroup.Name = newName
	aGroup.Description = newDescription
	aGroup.Metadata = clients.Metadata{"field": "value2"}
	aChannel := channel
	aChannel.Name = newName
	aChannel.Description = newDescription
	aChannel.Metadata = sdk.Metadata{"field": "value2"}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelReq      sdk.Channel
		updateGroupReq  channels.Channel
		svcRes          channels.Channel
		svcErr          error
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:     "update channel name",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: newName,
			},
			updateGroupReq: channels.Channel{
				ID:   group.ID,
				Name: newName,
			},
			svcRes:   nGroup,
			svcErr:   nil,
			response: nChannel,
			err:      nil,
		},
		{
			desc:     "update channel description",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:          channel.ID,
				Description: newDescription,
			},
			updateGroupReq: channels.Channel{
				ID:          group.ID,
				Description: newDescription,
			},
			svcRes:   dGroup,
			svcErr:   nil,
			response: dChannel,
			err:      nil,
		},
		{
			desc:     "update channel metadata",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			updateGroupReq: channels.Channel{
				ID:       group.ID,
				Metadata: clients.Metadata{"field": "value2"},
			},
			svcRes:   mGroup,
			svcErr:   nil,
			response: mChannel,
			err:      nil,
		},
		{
			desc:     "update channel with every field defined",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:          channel.ID,
				Name:        newName,
				Description: newDescription,
				Metadata:    sdk.Metadata{"field": "value2"},
			},
			updateGroupReq: channels.Channel{
				ID:          group.ID,
				Name:        newName,
				Description: newDescription,
				Metadata:    clients.Metadata{"field": "value2"},
			},
			svcRes:   aGroup,
			svcErr:   nil,
			response: aChannel,
			err:      nil,
		},
		{
			desc:     "update channel name with invalid channel id",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:   wrongID,
				Name: newName,
			},
			updateGroupReq: channels.Channel{
				ID:   wrongID,
				Name: newName,
			},
			svcRes:   channels.Channel{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Channel{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update channel description with invalid channel id",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:          wrongID,
				Description: newDescription,
			},
			updateGroupReq: channels.Channel{
				ID:          wrongID,
				Description: newDescription,
			},
			svcRes:   channels.Channel{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Channel{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update channel metadata with invalid channel id",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID: wrongID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			updateGroupReq: channels.Channel{
				ID:       wrongID,
				Metadata: clients.Metadata{"field": "value2"},
			},
			svcRes:   channels.Channel{},
			svcErr:   svcerr.ErrNotFound,
			response: sdk.Channel{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:     "update channel with invalid token",
			domainID: domainID,
			token:    invalidToken,
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: newName,
			},
			updateGroupReq: channels.Channel{
				ID:   group.ID,
				Name: newName,
			},
			svcRes:          channels.Channel{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Channel{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "update channel with empty token",
			domainID: domainID,
			token:    "",
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: newName,
			},
			updateGroupReq: channels.Channel{
				ID:   group.ID,
				Name: newName,
			},
			svcRes:   channels.Channel{},
			svcErr:   nil,
			response: sdk.Channel{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "update channel with name that is too long",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: strings.Repeat("a", 1025),
			},
			updateGroupReq: channels.Channel{},
			svcRes:         channels.Channel{},
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc:     "update channel that can't be marshalled",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			updateGroupReq: channels.Channel{},
			svcRes:         channels.Channel{},
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "update channel with service response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				ID:   channel.ID,
				Name: newName,
			},
			updateGroupReq: channels.Channel{
				ID:   group.ID,
				Name: newName,
			},
			svcRes: channels.Channel{
				ID: generateUUID(t),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Channel{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
		{
			desc:     "update channel with empty channel id",
			domainID: domainID,
			token:    validToken,
			channelReq: sdk.Channel{
				Name: newName,
			},
			updateGroupReq: channels.Channel{},
			svcRes:         channels.Channel{},
			svcErr:         nil,
			response:       sdk.Channel{},
			err:            errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("UpdateGroup", mock.Anything, tc.session, tc.updateGroupReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.UpdateChannel(tc.channelReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "UpdateGroup", mock.Anything, tc.session, tc.updateGroupReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListChannelsByClient(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	nChannels := uint64(10)
	aChannels := []sdk.Channel{}

	for i := uint64(1); i < nChannels; i++ {
		channel := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("membership_%d@example.com", i),
			Metadata: sdk.Metadata{"role": "channel"},
			Status:   clients.EnabledStatus.String(),
		}
		aChannels = append(aChannels, channel)
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		ClientID        string
		pageMeta        sdk.PageMetadata
		listGroupsReq   channels.Page
		svcRes          channels.Page
		svcErr          error
		authenticateErr error
		response        sdk.ChannelsPage
		err             errors.SDKError
	}{
		{
			desc:     "list channels successfully",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: nChannels,
				},
				Channels: convertChannels(aChannels),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: nChannels,
				},
				Channels: aChannels,
			},
			err: nil,
		},
		{
			desc:     "list channel with offset and limit",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 6,
				Limit:  nChannels,
			},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 6,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: uint64(len(aChannels[6 : nChannels-1])),
				},
				Channels: convertChannels(aChannels[6 : nChannels-1]),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(aChannels[6 : nChannels-1])),
				},
				Channels: aChannels[6 : nChannels-1],
			},
			err: nil,
		},
		{
			desc:     "list channel with given name",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Name:   "membership_8@example.com",
				Offset: 0,
				Limit:  nChannels,
			},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Name:   "membership_8@example.com",
					Offset: 0,
					Limit:  nChannels,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: convertChannels([]sdk.Channel{aChannels[8]}),
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Channels: aChannels[8:9],
			},
			err: nil,
		},
		{
			desc:     "list channels with invalid token",
			domainID: domainID,
			token:    invalidToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:          channels.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ChannelsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:          "list channels with empty token",
			domainID:      domainID,
			token:         "",
			ClientID:      testsutil.GenerateUUID(t),
			pageMeta:      sdk.PageMetadata{},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.ChannelsPage{},
			err:           errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list channels with limit greater than max",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Limit: 110,
			},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.ChannelsPage{},
			err:           errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:     "list channels with invalid metadata",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Metadata: sdk.Metadata{
					"test": make(chan int),
				},
			},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.ChannelsPage{},
			err:           errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "list channels with service response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			ClientID: testsutil.GenerateUUID(t),
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
				Channels: []channels.Channel{{
					ID: generateUUID(t),
					Metadata: clients.Metadata{
						"test": make(chan int),
					},
				}},
			},
			svcErr:   nil,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.ClientsKind, tc.ClientID, tc.listGroupsReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ChannelsByClient(tc.ClientID, tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.ClientsKind, tc.ClientID, tc.listGroupsReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestEnableChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	group := convertChannel(channel)
	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		svcRes          channels.Channel
		svcErr          error
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:      "enable channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcRes:    group,
			svcErr:    nil,
			response:  channel,
			err:       nil,
		},
		{
			desc:            "enable channel with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			svcRes:          channels.Channel{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Channel{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "enable channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "enable channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrNotFound,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:      "enable channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "enable channel with service response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcRes: channels.Channel{
				ID: generateUUID(t),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Channel{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("EnableGroup", mock.Anything, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.EnableChannel(tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "EnableGroup", mock.Anything, tc.session, tc.channelID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisableChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	group := convertChannel(channel)
	dGroup := group
	dGroup.Status = clients.DisabledStatus
	dChannel := channel
	dChannel.Status = clients.DisabledStatus.String()

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		svcRes          channels.Channel
		svcErr          error
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:      "disable channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcRes:    dGroup,
			svcErr:    nil,
			response:  dChannel,
			err:       nil,
		},
		{
			desc:            "disable channel with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			svcRes:          channels.Channel{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Channel{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "disable channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "disable channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrNotFound,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc:      "disable channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			svcRes:    channels.Channel{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "disable channel with service response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcRes: channels.Channel{
				ID: generateUUID(t),
				Metadata: clients.Metadata{
					"test": make(chan int),
				},
			},
			svcErr:   nil,
			response: sdk.Channel{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("DisableGroup", mock.Anything, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.DisableChannel(tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DisableGroup", mock.Anything, tc.session, tc.channelID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDeleteChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:      "delete channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:            "delete channel with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "delete channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "delete channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			svcErr:    svcerr.ErrRemoveEntity,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrRemoveEntity, http.StatusUnprocessableEntity),
		},
		{
			desc:      "delete channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			svcErr:    svcerr.ErrRemoveEntity,
			err:       errors.NewSDKError(apiutil.ErrMissingID),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("DeleteGroup", mock.Anything, tc.session, tc.channelID).Return(tc.svcErr)
			err := mgsdk.DeleteChannel(tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "DeleteGroup", mock.Anything, tc.session, tc.channelID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestChannelPermissions(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		svcRes          []string
		svcErr          error
		authenticateErr error
		response        sdk.Channel
		err             errors.SDKError
	}{
		{
			desc:      "view channel permissions successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			svcRes:    []string{"view"},
			svcErr:    nil,
			response: sdk.Channel{
				Permissions: []string{"view"},
			},
			err: nil,
		},
		{
			desc:            "view channel permissions with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			svcRes:          []string{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.Channel{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "view channel permissions with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			svcRes:    []string{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "view channel permissions with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			svcRes:    []string{},
			svcErr:    svcerr.ErrAuthorization,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "view channel permissions with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			svcRes:    []string{},
			svcErr:    nil,
			response:  sdk.Channel{},
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ViewGroupPerms", mock.Anything, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ChannelPermissions(tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ViewGroupPerms", mock.Anything, tc.session, tc.channelID)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddUserToChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		addUserReq      sdk.UsersRelationRequest
		authenticateErr error
		svcErr          error
		err             errors.SDKError
	}{
		{
			desc:      "add user to channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:      "add user to channel with invalid token",
			domainID:  domainID,
			token:     invalidToken,
			channelID: channel.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "add user to channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "add user to channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "add user to channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "add users to channel with empty relation",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingRelation), http.StatusBadRequest),
		},
		{
			desc:      "add users to channel with empty user ids",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			addUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.channelID, tc.addUserReq.Relation, policies.UsersKind, tc.addUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.AddUserToChannel(tc.channelID, tc.addUserReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.session, tc.channelID, tc.addUserReq.Relation, policies.UsersKind, tc.addUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveUserFromChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		removeUserReq   sdk.UsersRelationRequest
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:      "remove user from channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:      "remove user from channel with invalid token",
			domainID:  domainID,
			token:     invalidToken,
			channelID: channel.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "remove user from channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "remove user from channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "remove user from channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{user.ID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "remove users from channel with empty user ids",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			removeUserReq: sdk.UsersRelationRequest{
				Relation: "member",
				UserIDs:  []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.channelID, tc.removeUserReq.Relation, policies.UsersKind, tc.removeUserReq.UserIDs).Return(tc.svcErr)
			err := mgsdk.RemoveUserFromChannel(tc.channelID, tc.removeUserReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.session, tc.channelID, tc.removeUserReq.Relation, policies.UsersKind, tc.removeUserReq.UserIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestAddUserGroupToChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	relation := "parent_group"

	groupID := generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		addUserGroupReq sdk.UserGroupsRequest
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:      "add user group to channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:      "add user group to channel with invalid token",
			domainID:  domainID,
			token:     invalidToken,
			channelID: channel.ID,
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "add user group to channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "add user group to channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "add user group to channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "add user group to channel with empty group ids",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			addUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.channelID, relation, policies.ChannelsKind, tc.addUserGroupReq.UserGroupIDs).Return(tc.svcErr)
			err := mgsdk.AddUserGroupToChannel(tc.channelID, tc.addUserGroupReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.session, tc.channelID, relation, policies.ChannelsKind, tc.addUserGroupReq.UserGroupIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestRemoveUserGroupFromChannel(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	relation := "parent_group"

	groupID := generateUUID(t)

	cases := []struct {
		desc               string
		domainID           string
		token              string
		session            mgauthn.Session
		channelID          string
		removeUserGroupReq sdk.UserGroupsRequest
		svcErr             error
		authenticateErr    error
		err                errors.SDKError
	}{
		{
			desc:      "remove user group from channel successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:      "remove user group from channel with invalid token",
			domainID:  domainID,
			token:     invalidToken,
			channelID: channel.ID,
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "remove user group from channel with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "remove user group from channel with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "remove user group from channel with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{groupID},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "remove user group from channel with empty group ids",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			removeUserGroupReq: sdk.UserGroupsRequest{
				UserGroupIDs: []string{},
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrEmptyList), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.channelID, relation, policies.ChannelsKind, tc.removeUserGroupReq.UserGroupIDs).Return(tc.svcErr)
			err := mgsdk.RemoveUserGroupFromChannel(tc.channelID, tc.removeUserGroupReq, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.session, tc.channelID, relation, policies.ChannelsKind, tc.removeUserGroupReq.UserGroupIDs)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListChannelUserGroups(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		UsersURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	nGroups := uint64(10)
	aGroups := []sdk.Channel{}

	for i := uint64(1); i < nGroups; i++ {
		group := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("group_%d", i),
			Metadata: sdk.Metadata{"role": "group"},
			Status:   clients.EnabledStatus.String(),
		}
		aGroups = append(aGroups, group)
	}

	cases := []struct {
		desc            string
		token           string
		domainID        string
		session         mgauthn.Session
		channelID       string
		pageMeta        sdk.PageMetadata
		listGroupsReq   channels.Page
		svcRes          channels.Page
		svcErr          error
		authenticateErr error
		response        sdk.GroupsPage
		err             errors.SDKError
	}{
		{
			desc:      "list user channels successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			pageMeta:  sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: nGroups,
				},
			},
			svcErr: nil,
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: nGroups,
				},
			},
			err: nil,
		},
		{
			desc:      "list user channels with offset and limit",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 6,
				Limit:  nGroups,
			},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 6,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: uint64(len(aGroups[6 : nGroups-1])),
				},
			},
			svcErr: nil,
			response: sdk.GroupsPage{
				PageRes: sdk.PageRes{
					Total: uint64(len(aGroups[6 : nGroups-1])),
				},
			},
			err: nil,
		},
		{
			desc:      "list user channels with invalid token",
			domainID:  domainID,
			token:     invalidToken,
			channelID: channel.ID,
			pageMeta:  sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:          channels.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.GroupsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "list user channels with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			pageMeta:  sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:   channels.Page{},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "list user channels with limit greater than max",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			pageMeta: sdk.PageMetadata{
				Limit: 110,
			},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.GroupsPage{},
			err:           errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
		},
		{
			desc:      "list user channels with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			pageMeta: sdk.PageMetadata{
				DomainID: domainID,
			},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:   channels.Page{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "list users channels with level exceeding max",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			pageMeta: sdk.PageMetadata{
				Level: 10,
			},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.GroupsPage{},
			err:           errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrInvalidLevel), http.StatusBadRequest),
		},
		{
			desc:      "list users with invalid page metadata",
			token:     validToken,
			channelID: channel.ID,
			pageMeta: sdk.PageMetadata{
				Offset:   0,
				Limit:    10,
				DomainID: domainID,
				Metadata: sdk.Metadata{
					"test": make(chan int),
				},
			},
			listGroupsReq: channels.Page{},
			svcRes:        channels.Page{},
			svcErr:        nil,
			response:      sdk.GroupsPage{},
			err:           errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:      "list user channels with service response that can't be unmarshalled",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			pageMeta:  sdk.PageMetadata{},
			listGroupsReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
			},
			svcErr:   nil,
			response: sdk.GroupsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.ChannelsKind, tc.channelID, tc.listGroupsReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListChannelUserGroups(tc.channelID, tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.ChannelsKind, tc.channelID, tc.listGroupsReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestConnect(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	clientID := generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		connection      sdk.Connection
		svcErr          error
		authenticateRes mgauthn.Session
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "connect successfully",
			domainID: domainID,
			token:    validToken,
			connection: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "connect with invalid token",
			domainID: domainID,
			token:    invalidToken,
			connection: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "connect with empty token",
			domainID: domainID,
			token:    "",
			connection: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			err: errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "connect with invalid channel id",
			domainID: domainID,
			token:    validToken,
			connection: sdk.Connection{
				ChannelID: wrongID,
				ClientID:  clientID,
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "connect with empty channel id",
			domainID: domainID,
			token:    validToken,
			connection: sdk.Connection{
				ChannelID: "",
				ClientID:  clientID,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "connect with empty client id",
			domainID: domainID,
			token:    validToken,
			connection: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  "",
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.connection.ChannelID, policies.GroupRelation, policies.ClientsKind, []string{tc.connection.ClientID}).Return(tc.svcErr)
			err := mgsdk.Connect(tc.connection, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.session, tc.connection.ChannelID, policies.GroupRelation, policies.ClientsKind, []string{tc.connection.ClientID})
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnect(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	clientID := generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		disconnect      sdk.Connection
		svcErr          error
		authenticateRes mgauthn.Session
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:     "disconnect successfully",
			domainID: domainID,
			token:    validToken,
			disconnect: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			svcErr: nil,
			err:    nil,
		},
		{
			desc:     "disconnect with invalid token",
			domainID: domainID,
			token:    invalidToken,
			disconnect: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "disconnect with empty token",
			domainID: domainID,
			token:    "",
			disconnect: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  clientID,
			},
			err: errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "disconnect with invalid channel id",
			domainID: domainID,
			token:    validToken,
			disconnect: sdk.Connection{
				ChannelID: wrongID,
				ClientID:  clientID,
			},
			svcErr: svcerr.ErrAuthorization,
			err:    errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "disconnect with empty channel id",
			domainID: domainID,
			token:    validToken,
			disconnect: sdk.Connection{
				ChannelID: "",
				ClientID:  clientID,
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:     "disconnect with empty client id",
			domainID: domainID,
			token:    validToken,
			disconnect: sdk.Connection{
				ChannelID: channel.ID,
				ClientID:  "",
			},
			svcErr: nil,
			err:    errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.disconnect.ChannelID, policies.GroupRelation, policies.ClientsKind, []string{tc.disconnect.ClientID}).Return(tc.svcErr)
			err := mgsdk.Disconnect(tc.disconnect, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.session, tc.disconnect.ChannelID, policies.GroupRelation, policies.ClientsKind, []string{tc.disconnect.ClientID})
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestConnectClient(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	clientID := generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		clientID        string
		svcErr          error
		authenticateRes mgauthn.Session
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:      "connect successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			clientID:  clientID,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:            "connect with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			clientID:        clientID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "connect with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			clientID:  clientID,
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "connect with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			clientID:  clientID,
			svcErr:    svcerr.ErrAuthorization,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "connect with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			clientID:  clientID,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "connect with empty client id",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			clientID:  "",
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Assign", mock.Anything, tc.session, tc.channelID, policies.GroupRelation, policies.ClientsKind, []string{tc.clientID}).Return(tc.svcErr)
			err := mgsdk.ConnectClient(tc.clientID, tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Assign", mock.Anything, tc.session, tc.channelID, policies.GroupRelation, policies.ClientsKind, []string{tc.clientID})
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestDisconnectClient(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	clientID := generateUUID(t)

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		channelID       string
		clientID        string
		svcErr          error
		authenticateErr error
		err             errors.SDKError
	}{
		{
			desc:      "disconnect successfully",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			clientID:  clientID,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:            "disconnect with invalid token",
			domainID:        domainID,
			token:           invalidToken,
			channelID:       channel.ID,
			clientID:        clientID,
			authenticateErr: svcerr.ErrAuthentication,
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:      "disconnect with empty token",
			domainID:  domainID,
			token:     "",
			channelID: channel.ID,
			clientID:  clientID,
			err:       errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:      "disconnect with invalid channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: wrongID,
			clientID:  clientID,
			svcErr:    svcerr.ErrAuthorization,
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:      "disconnect with empty channel id",
			domainID:  domainID,
			token:     validToken,
			channelID: "",
			clientID:  clientID,
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
		{
			desc:      "disconnect with empty client id",
			domainID:  domainID,
			token:     validToken,
			channelID: channel.ID,
			clientID:  "",
			svcErr:    nil,
			err:       errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrMissingID), http.StatusBadRequest),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("Unassign", mock.Anything, tc.session, tc.channelID, policies.GroupRelation, policies.ClientsKind, []string{tc.clientID}).Return(tc.svcErr)
			err := mgsdk.DisconnectClient(tc.clientID, tc.channelID, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "Unassign", mock.Anything, tc.session, tc.channelID, policies.GroupRelation, policies.ClientsKind, []string{tc.clientID})
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func TestListGroupChannels(t *testing.T) {
	ts, gsvc, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ClientsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	groupChannel := sdk.Channel{
		ID:       testsutil.GenerateUUID(t),
		Name:     "group_channel",
		Metadata: sdk.Metadata{"role": "group"},
		Status:   clients.EnabledStatus.String(),
	}

	cases := []struct {
		desc            string
		domainID        string
		token           string
		session         mgauthn.Session
		groupID         string
		pageMeta        sdk.PageMetadata
		svcReq          channels.Page
		svcRes          channels.Page
		svcErr          error
		authenticateErr error
		response        sdk.ChannelsPage
		err             errors.SDKError
	}{
		{
			desc:     "list group channels successfully",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
			},
			svcErr: nil,
			response: sdk.ChannelsPage{
				PageRes: sdk.PageRes{
					Total: 1,
				},
				Channels: []sdk.Channel{groupChannel},
			},
			err: nil,
		},
		{
			desc:     "list group channels with invalid token",
			domainID: domainID,
			token:    invalidToken,
			groupID:  group.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:          channels.Page{},
			authenticateErr: svcerr.ErrAuthentication,
			response:        sdk.ChannelsPage{},
			err:             errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
		{
			desc:     "list group channels with empty token",
			domainID: domainID,
			token:    "",
			groupID:  group.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq:   channels.Page{},
			svcRes:   channels.Page{},
			svcErr:   nil,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKErrorWithStatus(apiutil.ErrBearerToken, http.StatusUnauthorized),
		},
		{
			desc:     "list group channels with invalid group id",
			domainID: domainID,
			token:    validToken,
			groupID:  wrongID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes:   channels.Page{},
			svcErr:   svcerr.ErrAuthorization,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc:     "list group channels with invalid page metadata",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
				Metadata: sdk.Metadata{
					"test": make(chan int),
				},
			},
			svcReq:   channels.Page{},
			svcRes:   channels.Page{},
			svcErr:   nil,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKError(errors.New("json: unsupported type: chan int")),
		},
		{
			desc:     "list group channels with service response that can't be unmarshalled",
			domainID: domainID,
			token:    validToken,
			groupID:  group.ID,
			pageMeta: sdk.PageMetadata{
				Offset: 0,
				Limit:  10,
			},
			svcReq: channels.Page{
				PageMetadata: channels.PageMetadata{
					Offset: 0,
					Limit:  10,
				},
			},
			svcRes: channels.Page{
				PageMetadata: channels.PageMetadata{
					Total: 1,
				},
			},
			svcErr:   nil,
			response: sdk.ChannelsPage{},
			err:      errors.NewSDKError(errors.New("unexpected end of JSON input")),
		},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.token == validToken {
				tc.session = mgauthn.Session{DomainUserID: domainID + "_" + validID, UserID: validID, DomainID: domainID}
			}
			authCall := auth.On("Authenticate", mock.Anything, tc.token).Return(tc.session, tc.authenticateErr)
			svcCall := gsvc.On("ListGroups", mock.Anything, tc.session, policies.GroupsKind, tc.groupID, tc.svcReq).Return(tc.svcRes, tc.svcErr)
			resp, err := mgsdk.ListGroupChannels(tc.groupID, tc.pageMeta, tc.domainID, tc.token)
			assert.Equal(t, tc.err, err)
			assert.Equal(t, tc.response, resp)
			if tc.err == nil {
				ok := svcCall.Parent.AssertCalled(t, "ListGroups", mock.Anything, tc.session, policies.GroupsKind, tc.groupID, tc.svcReq)
				assert.True(t, ok)
			}
			svcCall.Unset()
			authCall.Unset()
		})
	}
}

func generateTestChannel(t *testing.T) sdk.Channel {
	createdAt, err := time.Parse(time.RFC3339, "2023-03-03T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	updatedAt := createdAt
	ch := sdk.Channel{
		ID:          testsutil.GenerateUUID(&testing.T{}),
		DomainID:    testsutil.GenerateUUID(&testing.T{}),
		Name:        channelName,
		Description: description,
		Metadata:    sdk.Metadata{"role": "client"},
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		Status:      clients.EnabledStatus.String(),
	}
	return ch
}
