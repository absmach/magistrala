// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/internal/apiutil"
	"github.com/absmach/magistrala/internal/groups"
	"github.com/absmach/magistrala/internal/testsutil"
	mglog "github.com/absmach/magistrala/logger"
	mgclients "github.com/absmach/magistrala/pkg/clients"
	"github.com/absmach/magistrala/pkg/errors"
	repoerr "github.com/absmach/magistrala/pkg/errors/repository"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/groups/mocks"
	sdk "github.com/absmach/magistrala/pkg/sdk/go"
	"github.com/absmach/magistrala/things"
	api "github.com/absmach/magistrala/things/api/http"
	thmocks "github.com/absmach/magistrala/things/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupChannels() (*httptest.Server, *mocks.Repository, *authmocks.AuthClient) {
	cRepo := new(thmocks.Repository)
	grepo := new(mocks.Repository)
	thingCache := new(thmocks.Cache)

	auth := new(authmocks.AuthClient)
	csvc := things.NewService(auth, cRepo, grepo, thingCache, idProvider)
	gsvc := groups.NewService(grepo, idProvider, auth)

	logger := mglog.NewMock()
	mux := chi.NewRouter()
	api.MakeHandler(csvc, gsvc, mux, logger, "")

	return httptest.NewServer(mux), grepo, auth
}

func TestCreateChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	channel := sdk.Channel{
		Name:     "channelName",
		Metadata: validMetadata,
		Status:   mgclients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     errors.SDKError
	}{
		{
			desc:    "create channel successfully",
			channel: channel,
			token:   token,
			err:     nil,
		},
		{
			desc:    "create channel with existing name",
			channel: channel,
			token:   token,
			err:     nil,
		},
		{
			desc: "update channel that can't be marshalled",
			channel: sdk.Channel{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			token: token,
			err:   errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
		{
			desc: "create channel with parent",
			channel: sdk.Channel{
				Name:     gName,
				ParentID: testsutil.GenerateUUID(t),
				Status:   mgclients.EnabledStatus.String(),
			},
			token: token,
			err:   nil,
		},
		{
			desc: "create channel with invalid parent",
			channel: sdk.Channel{
				Name:     gName,
				ParentID: wrongID,
				Status:   mgclients.EnabledStatus.String(),
			},
			token: token,
			err:   errors.NewSDKErrorWithStatus(svcerr.ErrCreateEntity, http.StatusUnprocessableEntity),
		},
		{
			desc: "create channel with missing name",
			channel: sdk.Channel{
				Status: mgclients.EnabledStatus.String(),
			},
			token: token,
			err:   errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrNameSize), http.StatusBadRequest),
		},
		{
			desc: "create a channel with every field defined",
			channel: sdk.Channel{
				ID:          generateUUID(t),
				ParentID:    "parent",
				Name:        "name",
				Description: description,
				Metadata:    validMetadata,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Status:      mgclients.EnabledStatus.String(),
			},
			token: token,
			err:   nil,
		},
	}
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		authCall1 := auth.On("AddPolicies", mock.Anything, mock.Anything).Return(&magistrala.AddPoliciesRes{Added: true}, nil)
		authCall2 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		authCall3 := auth.On("DeletePolicies", mock.Anything, mock.Anything).Return(&magistrala.DeletePoliciesRes{Deleted: false}, nil)
		repoCall := grepo.On("Save", mock.Anything, mock.Anything).Return(convertChannel(sdk.Channel{}), tc.err)
		rChannel, err := mgsdk.CreateChannel(tc.channel, validToken)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		if err == nil {
			assert.NotEmpty(t, rChannel, fmt.Sprintf("%s: expected not nil on client ID", tc.desc))
			ok := repoCall.Parent.AssertCalled(t, "Save", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Save was not called on %s", tc.desc))
		}
		authCall.Unset()
		authCall1.Unset()
		authCall2.Unset()
		authCall3.Unset()
		repoCall.Unset()
	}
}

func TestListChannels(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	var chs []sdk.Channel
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		gr := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("channel_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
			Status:   mgclients.EnabledStatus.String(),
		}
		chs = append(chs, gr)
	}

	cases := []struct {
		desc     string
		token    string
		status   mgclients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Channel
		ctx      context.Context
	}{
		{
			desc:     "get a list of channels",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			err:      nil,
			response: chs[offset:limit],
		},
		{
			desc:     "get a list of channels with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of channels with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of channels with zero limit",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      nil,
			response: nil,
		},
		{
			desc:     "get a list of channels with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Channel(nil),
		},
		{
			desc:     "get a list of channels with given name",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
		{
			desc:     "get a list of channels with level",
			token:    token,
			offset:   0,
			limit:    1,
			level:    1,
			err:      nil,
			response: []sdk.Channel{chs[0]},
		},
		{
			desc:     "get a list of channels with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == invalidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: invalidToken}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertChannels(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
		}
		page, err := mgsdk.Channels(pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Channels), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestListUserChannels(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	var chs []sdk.Channel
	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	for i := 10; i < 100; i++ {
		gr := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("channel_%d", i),
			Metadata: sdk.Metadata{"name": fmt.Sprintf("thing_%d", i)},
			Status:   mgclients.EnabledStatus.String(),
		}
		chs = append(chs, gr)
	}

	cases := []struct {
		desc     string
		token    string
		status   mgclients.Status
		total    uint64
		offset   uint64
		limit    uint64
		level    int
		name     string
		userID   string
		metadata sdk.Metadata
		err      errors.SDKError
		response []sdk.Channel
		ctx      context.Context
	}{
		{
			desc:     "get a list of user channels",
			token:    token,
			limit:    limit,
			offset:   offset,
			total:    total,
			userID:   validID,
			err:      nil,
			response: chs[offset:limit],
		},
		{
			desc:     "get a list of user channels with invalid token",
			token:    invalidToken,
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of user channels with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			userID:   validID,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerToken), http.StatusUnauthorized),
			response: nil,
		},
		{
			desc:     "get a list of user channels with limit greater than max",
			token:    token,
			userID:   validID,
			offset:   offset,
			limit:    110,
			err:      errors.NewSDKErrorWithStatus(errors.Wrap(apiutil.ErrValidation, apiutil.ErrLimitSize), http.StatusBadRequest),
			response: []sdk.Channel(nil),
		},
		{
			desc:     "get a list of user channels with given name",
			token:    token,
			offset:   0,
			limit:    1,
			userID:   validID,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
		{
			desc:     "get a list of user channels with level",
			token:    token,
			offset:   0,
			limit:    1,
			userID:   validID,
			level:    1,
			err:      nil,
			response: []sdk.Channel{chs[0]},
		},
		{
			desc:     "get a list of user channels with metadata",
			token:    token,
			offset:   0,
			limit:    1,
			userID:   validID,
			err:      nil,
			metadata: sdk.Metadata{},
			response: []sdk.Channel{chs[89]},
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		if tc.token == invalidToken {
			repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: invalidToken}).Return(&magistrala.IdentityRes{}, svcerr.ErrAuthentication)
			repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, svcerr.ErrAuthorization)
		}
		repoCall2 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertChannels(tc.response)}, tc.err)
		pm := sdk.PageMetadata{
			Offset: tc.offset,
			Limit:  tc.limit,
			Level:  uint64(tc.level),
			User:   tc.userID,
		}
		page, err := mgsdk.ListUserChannels(pm, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, len(tc.response), len(page.Channels), fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page))
		if tc.err == nil {
			ok := repoCall3.Parent.AssertCalled(t, "RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("RetrieveByIDs was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
	}
}

func TestViewChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	channel := sdk.Channel{
		Name:        "channelName",
		Description: description,
		Metadata:    validMetadata,
		Children:    []*sdk.Channel{},
		Status:      mgclients.EnabledStatus.String(),
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)
	channel.ID = generateUUID(t)

	cases := []struct {
		desc      string
		token     string
		channelID string
		response  sdk.Channel
		err       errors.SDKError
	}{
		{
			desc:      "view channel",
			token:     validToken,
			channelID: channel.ID,
			response:  channel,
			err:       nil,
		},
		{
			desc:      "view channel with invalid token",
			token:     "wrongtoken",
			channelID: channel.ID,
			response:  sdk.Channel{Children: []*sdk.Channel{}},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
		{
			desc:      "view channel for wrong id",
			token:     validToken,
			channelID: wrongID,
			response:  sdk.Channel{Children: []*sdk.Channel{}},
			err:       errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall1 := grepo.On("RetrieveByID", mock.Anything, tc.channelID).Return(convertChannel(tc.response), tc.err)
		grp, err := mgsdk.Channel(tc.channelID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if len(tc.response.Children) == 0 {
			tc.response.Children = nil
		}
		if len(grp.Children) == 0 {
			grp.Children = nil
		}
		assert.Equal(t, tc.response, grp, fmt.Sprintf("%s: expected metadata %v got %v\n", tc.desc, tc.response, grp))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, tc.channelID)
			assert.True(t, ok, fmt.Sprintf("RetrieveByID was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	channel := sdk.Channel{
		ID:          generateUUID(t),
		Name:        "channelsName",
		Description: description,
		Metadata:    validMetadata,
	}

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	channel.ID = generateUUID(t)

	cases := []struct {
		desc     string
		token    string
		channel  sdk.Channel
		response sdk.Channel
		err      errors.SDKError
	}{
		{
			desc: "update channel name",
			channel: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			response: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update channel description",
			channel: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			response: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update channel metadata",
			channel: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			token: validToken,
			err:   nil,
		},
		{
			desc: "update channel name with invalid channel id",
			channel: sdk.Channel{
				ID:   wrongID,
				Name: "NewName",
			},
			response: sdk.Channel{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel description with invalid channel id",
			channel: sdk.Channel{
				ID:          wrongID,
				Description: "NewDescription",
			},
			response: sdk.Channel{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel metadata with invalid channel id",
			channel: sdk.Channel{
				ID: wrongID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{},
			token:    validToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrNotFound, http.StatusNotFound),
		},
		{
			desc: "update channel name with invalid token",
			channel: sdk.Channel{
				ID:   channel.ID,
				Name: "NewName",
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update channel description with invalid token",
			channel: sdk.Channel{
				ID:          channel.ID,
				Description: "NewDescription",
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update channel metadata with invalid token",
			channel: sdk.Channel{
				ID: channel.ID,
				Metadata: sdk.Metadata{
					"field": "value2",
				},
			},
			response: sdk.Channel{},
			token:    invalidToken,
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden),
		},
		{
			desc: "update channel that can't be marshalled",
			channel: sdk.Channel{
				Name: "test",
				Metadata: map[string]interface{}{
					"test": make(chan int),
				},
			},
			response: sdk.Channel{},
			token:    token,
			err:      errors.NewSDKError(fmt.Errorf("json: unsupported type: chan int")),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall1 := grepo.On("Update", mock.Anything, mock.Anything).Return(convertChannel(tc.response), tc.err)
		_, err := mgsdk.UpdateChannel(tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			ok := repoCall1.Parent.AssertCalled(t, "Update", mock.Anything, mock.Anything)
			assert.True(t, ok, fmt.Sprintf("Update was not called on %s", tc.desc))
		}
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestListChannelsByThing(t *testing.T) {
	ts, grepo, auth := setupChannels()
	auth.Test(t)
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	nChannels := uint64(10)
	aChannels := []sdk.Channel{}

	for i := uint64(1); i < nChannels; i++ {
		channel := sdk.Channel{
			ID:       generateUUID(t),
			Name:     fmt.Sprintf("membership_%d@example.com", i),
			Metadata: sdk.Metadata{"role": "channel"},
			Status:   mgclients.EnabledStatus.String(),
		}
		aChannels = append(aChannels, channel)
	}

	cases := []struct {
		desc     string
		token    string
		page     sdk.PageMetadata
		response []sdk.Channel
		err      errors.SDKError
	}{
		{
			desc:  "list channel with authorized token",
			token: validToken,
			page: sdk.PageMetadata{
				Thing: testsutil.GenerateUUID(t),
			},
			response: aChannels,
			err:      nil,
		},
		{
			desc:  "list channel with offset and limit",
			token: validToken,
			page: sdk.PageMetadata{
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mgclients.AllStatus.String(),
				Thing:  testsutil.GenerateUUID(t),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:  "list channel with given name",
			token: validToken,
			page: sdk.PageMetadata{
				Name:   gName,
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mgclients.AllStatus.String(),
				Thing:  testsutil.GenerateUUID(t),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:  "list channel with given level",
			token: validToken,
			page: sdk.PageMetadata{
				Level:  1,
				Offset: 6,
				Total:  nChannels,
				Limit:  nChannels,
				Status: mgclients.AllStatus.String(),
				Thing:  testsutil.GenerateUUID(t),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:  "list channel with metadata",
			token: validToken,
			page: sdk.PageMetadata{
				Metadata: validMetadata,
				Offset:   6,
				Total:    nChannels,
				Limit:    nChannels,
				Status:   mgclients.AllStatus.String(),
				Thing:    testsutil.GenerateUUID(t),
			},
			response: aChannels[6 : nChannels-1],
			err:      nil,
		},
		{
			desc:  "list channel with an invalid token",
			token: invalidToken,
			page: sdk.PageMetadata{
				Thing: testsutil.GenerateUUID(t),
			},
			response: []sdk.Channel(nil),
			err:      errors.NewSDKErrorWithStatus(svcerr.ErrAuthentication, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 := auth.On("ListAllSubjects", mock.Anything, mock.Anything).Return(&magistrala.ListSubjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall3 := auth.On("ListAllObjects", mock.Anything, mock.Anything).Return(&magistrala.ListObjectsRes{Policies: toIDs(tc.response)}, nil)
		repoCall4 := grepo.On("RetrieveByIDs", mock.Anything, mock.Anything, mock.Anything).Return(mggroups.Page{Groups: convertChannels(tc.response)}, tc.err)
		page, err := mgsdk.ChannelsByThing(tc.page, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Channels, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.response, page.Channels))
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
		repoCall3.Unset()
		repoCall4.Unset()
	}
}

func TestEnableChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	channel := sdk.Channel{
		ID:        generateUUID(t),
		Name:      gName,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mgclients.Disabled,
	}

	repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 := grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(mggroups.Group{}, repoerr.ErrNotFound)
	repoCall2 := grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(nil)
	_, err := mgsdk.EnableChannel("wrongID", validToken)
	assert.Equal(t, errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest), err, fmt.Sprintf("Enable channel with wrong id: expected %v got %v", svcerr.ErrViewEntity, err))
	ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "RetrieveByID was not called on enabling channel")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	ch := mggroups.Group{
		ID:        channel.ID,
		Name:      channel.Name,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mgclients.DisabledStatus,
	}
	repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 = grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(ch, nil)
	repoCall2 = grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(ch, nil)
	res, err := mgsdk.EnableChannel(channel.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Enable channel with correct id: expected %v got %v", nil, err))
	assert.Equal(t, channel, res, fmt.Sprintf("Enable channel with correct id: expected %v got %v", channel, res))
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, channel.ID)
	assert.True(t, ok, "RetrieveByID was not called on enabling channel")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on enabling channel")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestDisableChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	channel := sdk.Channel{
		ID:        generateUUID(t),
		Name:      gName,
		DomainID:  generateUUID(t),
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mgclients.Enabled,
	}

	repoCall := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 := grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(nil)
	repoCall2 := grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(mggroups.Group{}, repoerr.ErrNotFound)
	_, err := mgsdk.DisableChannel("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrViewEntity, http.StatusBadRequest), fmt.Sprintf("Disable channel with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	ok := repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, "wrongID")
	assert.True(t, ok, "Memberships was not called on disabling channel with wrong id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	ch := mggroups.Group{
		ID:        channel.ID,
		Name:      channel.Name,
		Domain:    channel.DomainID,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mgclients.EnabledStatus,
	}

	repoCall = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall1 = grepo.On("ChangeStatus", mock.Anything, mock.Anything).Return(ch, nil)
	repoCall2 = grepo.On("RetrieveByID", mock.Anything, mock.Anything).Return(ch, nil)
	res, err := mgsdk.DisableChannel(channel.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Disable channel with correct id: expected %v got %v", nil, err))
	assert.Equal(t, channel, res, fmt.Sprintf("Disable channel with correct id: expected %v got %v", channel, res))
	ok = repoCall1.Parent.AssertCalled(t, "RetrieveByID", mock.Anything, channel.ID)
	assert.True(t, ok, "RetrieveByID was not called on disabling channel with correct id")
	ok = repoCall2.Parent.AssertCalled(t, "ChangeStatus", mock.Anything, mock.Anything)
	assert.True(t, ok, "ChangeStatus was not called on disabling channel with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
}

func TestDeleteChannel(t *testing.T) {
	ts, grepo, auth := setupChannels()
	defer ts.Close()

	conf := sdk.Config{
		ThingsURL: ts.URL,
	}
	mgsdk := sdk.NewSDK(conf)

	creationTime := time.Now().UTC()
	channel := sdk.Channel{
		ID:        generateUUID(t),
		Name:      gName,
		CreatedAt: creationTime,
		UpdatedAt: creationTime,
		Status:    mgclients.Enabled,
	}

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall1 := auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: false}, nil)
	repoCall2 := grepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	err := mgsdk.DeleteChannel("wrongID", validToken)
	assert.Equal(t, err, errors.NewSDKErrorWithStatus(svcerr.ErrAuthorization, http.StatusForbidden), fmt.Sprintf("Delete channel with wrong id: expected %v got %v", svcerr.ErrNotFound, err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	repoCall = auth.On("DeletePolicy", mock.Anything, mock.Anything, mock.Anything).Return(&magistrala.DeletePolicyRes{Deleted: true}, nil)
	repoCall1 = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
	repoCall2 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall3 := grepo.On("Delete", mock.Anything, mock.Anything).Return(nil)
	err = mgsdk.DeleteChannel(channel.ID, validToken)
	assert.Nil(t, err, fmt.Sprintf("Delete channel with correct id: expected %v got %v", nil, err))
	ok := repoCall3.Parent.AssertCalled(t, "Delete", mock.Anything, channel.ID)
	assert.True(t, ok, "Delete was not called on deleting channel with correct id")
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()
	repoCall3.Unset()
}

func toIDs(objects interface{}) []string {
	v := reflect.ValueOf(objects)
	if v.Kind() != reflect.Slice {
		panic("objects argument must be a slice")
	}
	ids := make([]string, v.Len())
	for i := 0; i < v.Len(); i++ {
		id := v.Index(i).FieldByName("ID").String()
		ids[i] = id
	}

	return ids
}
