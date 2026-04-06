// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/channels"
	"github.com/absmach/magistrala/channels/events"
	"github.com/absmach/magistrala/channels/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/roles"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	storeClient  *redis.Client
	storeURL     string
	validSession = authn.Session{
		DomainID: testsutil.GenerateUUID(&testing.T{}),
		UserID:   testsutil.GenerateUUID(&testing.T{}),
	}
	validChannel      = generateTestChannel(&testing.T{})
	validChannelsPage = channels.ChannelsPage{
		Page: channels.Page{
			Limit:  10,
			Offset: 0,
			Total:  1,
		},
		Channels: []channels.Channel{validChannel},
	}
)

func newEventStoreMiddleware(t *testing.T) (*mocks.Service, channels.Service) {
	svc := new(mocks.Service)
	nsvc, err := events.NewEventStoreMiddleware(context.Background(), svc, storeURL)
	require.Nil(t, err, fmt.Sprintf("create events store middleware failed with unexpected error: %s", err))

	return svc, nsvc
}

func TestMain(m *testing.M) {
	code := testsutil.RunRedisTest(m, &storeClient, &storeURL)
	os.Exit(code)
}

func TestCreateChannels(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validID := testsutil.GenerateUUID(t)
	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, validID)

	cases := []struct {
		desc        string
		session     authn.Session
		channels    []channels.Channel
		svcRes      []channels.Channel
		svcRoleRes  []roles.RoleProvision
		svcErr      error
		resp        []channels.Channel
		respRoleRes []roles.RoleProvision
		err         error
	}{
		{
			desc:        "publish successfully",
			session:     validSession,
			channels:    []channels.Channel{validChannel},
			svcRes:      []channels.Channel{validChannel},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      nil,
			resp:        []channels.Channel{validChannel},
			respRoleRes: []roles.RoleProvision{},
			err:         nil,
		},
		{
			desc:        "failed to publish with service error",
			session:     validSession,
			channels:    []channels.Channel{validChannel},
			svcRes:      []channels.Channel{},
			svcRoleRes:  []roles.RoleProvision{},
			svcErr:      svcerr.ErrCreateEntity,
			resp:        []channels.Channel{},
			respRoleRes: []roles.RoleProvision{},
			err:         svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("CreateChannels", validCtx, tc.session, tc.channels).Return(tc.svcRes, tc.svcRoleRes, tc.svcErr)
			resp, respRoleRes, err := nsvc.CreateChannels(validCtx, tc.session, tc.channels...)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			assert.Equal(t, tc.respRoleRes, respRoleRes, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.respRoleRes, respRoleRes))
			svcCall.Unset()
		})
	}
}

func TestViewChannel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		channelID string
		withRoles bool
		svcRes    channels.Channel
		svcErr    error
		resp      channels.Channel
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			channelID: validChannel.ID,
			withRoles: false,
			svcRes:    validChannel,
			svcErr:    nil,
			resp:      validChannel,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			channelID: validChannel.ID,
			withRoles: false,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrViewEntity,
			resp:      channels.Channel{},
			err:       svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ViewChannel", validCtx, tc.session, tc.channelID, tc.withRoles).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ViewChannel(validCtx, tc.session, tc.channelID, tc.withRoles)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateChannel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedChannel := validChannel
	updatedChannel.Name = "updatedName"

	cases := []struct {
		desc    string
		session authn.Session
		channel channels.Channel
		svcRes  channels.Channel
		svcErr  error
		resp    channels.Channel
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			channel: updatedChannel,
			svcRes:  updatedChannel,
			svcErr:  nil,
			resp:    updatedChannel,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			channel: updatedChannel,
			svcRes:  channels.Channel{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    channels.Channel{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateChannel", validCtx, tc.session, tc.channel).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateChannel(validCtx, tc.session, tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestUpdateChannelTags(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	updatedChannel := validChannel
	updatedChannel.Tags = []string{"newTag1", "newTag2"}

	cases := []struct {
		desc    string
		session authn.Session
		channel channels.Channel
		svcRes  channels.Channel
		svcErr  error
		resp    channels.Channel
		err     error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			channel: updatedChannel,
			svcRes:  updatedChannel,
			svcErr:  nil,
			resp:    updatedChannel,
			err:     nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			channel: updatedChannel,
			svcRes:  channels.Channel{},
			svcErr:  svcerr.ErrUpdateEntity,
			resp:    channels.Channel{},
			err:     svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("UpdateChannelTags", validCtx, tc.session, tc.channel).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.UpdateChannelTags(validCtx, tc.session, tc.channel)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestEnableChannel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		channelID string
		svcRes    channels.Channel
		svcErr    error
		resp      channels.Channel
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			channelID: validChannel.ID,
			svcRes:    validChannel,
			svcErr:    nil,
			resp:      validChannel,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			channelID: validChannel.ID,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrUpdateEntity,
			resp:      channels.Channel{},
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("EnableChannel", validCtx, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.EnableChannel(validCtx, tc.session, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestDisableChannel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		channelID string
		svcRes    channels.Channel
		svcErr    error
		resp      channels.Channel
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			channelID: validChannel.ID,
			svcRes:    validChannel,
			svcErr:    nil,
			resp:      validChannel,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			channelID: validChannel.ID,
			svcRes:    channels.Channel{},
			svcErr:    svcerr.ErrUpdateEntity,
			resp:      channels.Channel{},
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("DisableChannel", validCtx, tc.session, tc.channelID).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.DisableChannel(validCtx, tc.session, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListChannels(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta channels.Page
		svcRes   channels.ChannelsPage
		svcErr   error
		resp     channels.ChannelsPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			pageMeta: channels.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validChannelsPage,
			svcErr: nil,
			resp:   validChannelsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			pageMeta: channels.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: channels.ChannelsPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   channels.ChannelsPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListChannels", validCtx, tc.session, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListChannels(validCtx, tc.session, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestListUserChannels(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc     string
		session  authn.Session
		userID   string
		pageMeta channels.Page
		svcRes   channels.ChannelsPage
		svcErr   error
		resp     channels.ChannelsPage
		err      error
	}{
		{
			desc:    "publish successfully",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: channels.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: validChannelsPage,
			svcErr: nil,
			resp:   validChannelsPage,
			err:    nil,
		},
		{
			desc:    "failed to publish with service error",
			session: validSession,
			userID:  validSession.UserID,
			pageMeta: channels.Page{
				Limit:  10,
				Offset: 0,
			},
			svcRes: channels.ChannelsPage{},
			svcErr: svcerr.ErrViewEntity,
			resp:   channels.ChannelsPage{},
			err:    svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("ListUserChannels", validCtx, tc.session, tc.userID, tc.pageMeta).Return(tc.svcRes, tc.svcErr)
			resp, err := nsvc.ListUserChannels(validCtx, tc.session, tc.userID, tc.pageMeta)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			assert.Equal(t, tc.resp, resp, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.resp, resp))
			svcCall.Unset()
		})
	}
}

func TestRemoveChannel(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		channelID string
		svcErr    error
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			channelID: validChannel.ID,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			channelID: validChannel.ID,
			svcErr:    svcerr.ErrRemoveEntity,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveChannel", validCtx, tc.session, tc.channelID).Return(tc.svcErr)
			err := nsvc.RemoveChannel(validCtx, tc.session, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestConnect(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		chIDs     []string
		clIDs     []string
		connTypes []connections.ConnType
		svcErr    error
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			chIDs:     []string{validChannel.ID},
			clIDs:     []string{testsutil.GenerateUUID(t)},
			connTypes: []connections.ConnType{connections.Publish},
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			chIDs:     []string{validChannel.ID},
			clIDs:     []string{testsutil.GenerateUUID(t)},
			connTypes: []connections.ConnType{connections.Publish},
			svcErr:    svcerr.ErrCreateEntity,
			err:       svcerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Connect", validCtx, tc.session, tc.chIDs, tc.clIDs, tc.connTypes).Return(tc.svcErr)
			err := nsvc.Connect(validCtx, tc.session, tc.chIDs, tc.clIDs, tc.connTypes)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestDisconnect(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		chIDs     []string
		clIDs     []string
		connTypes []connections.ConnType
		svcErr    error
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			chIDs:     []string{validChannel.ID},
			clIDs:     []string{testsutil.GenerateUUID(t)},
			connTypes: []connections.ConnType{connections.Publish},
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			chIDs:     []string{validChannel.ID},
			clIDs:     []string{testsutil.GenerateUUID(t)},
			connTypes: []connections.ConnType{connections.Publish},
			svcErr:    svcerr.ErrRemoveEntity,
			err:       svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("Disconnect", validCtx, tc.session, tc.chIDs, tc.clIDs, tc.connTypes).Return(tc.svcErr)
			err := nsvc.Disconnect(validCtx, tc.session, tc.chIDs, tc.clIDs, tc.connTypes)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestSetParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc          string
		session       authn.Session
		parentGroupID string
		channelID     string
		svcErr        error
		err           error
	}{
		{
			desc:          "publish successfully",
			session:       validSession,
			parentGroupID: testsutil.GenerateUUID(t),
			channelID:     validChannel.ID,
			svcErr:        nil,
			err:           nil,
		},
		{
			desc:          "failed to publish with service error",
			session:       validSession,
			parentGroupID: testsutil.GenerateUUID(t),
			channelID:     validChannel.ID,
			svcErr:        svcerr.ErrUpdateEntity,
			err:           svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("SetParentGroup", validCtx, tc.session, tc.parentGroupID, tc.channelID).Return(tc.svcErr)
			err := nsvc.SetParentGroup(validCtx, tc.session, tc.parentGroupID, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func TestRemoveParentGroup(t *testing.T) {
	svc, nsvc := newEventStoreMiddleware(t)

	validCtx := context.WithValue(context.Background(), middleware.RequestIDKey, testsutil.GenerateUUID(t))

	cases := []struct {
		desc      string
		session   authn.Session
		channelID string
		svcErr    error
		err       error
	}{
		{
			desc:      "publish successfully",
			session:   validSession,
			channelID: validChannel.ID,
			svcErr:    nil,
			err:       nil,
		},
		{
			desc:      "failed to publish with service error",
			session:   validSession,
			channelID: validChannel.ID,
			svcErr:    svcerr.ErrUpdateEntity,
			err:       svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			svcCall := svc.On("RemoveParentGroup", validCtx, tc.session, tc.channelID).Return(tc.svcErr)
			err := nsvc.RemoveParentGroup(validCtx, tc.session, tc.channelID)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			svcCall.Unset()
		})
	}
}

func generateTestChannel(t *testing.T) channels.Channel {
	createdAt, err := time.Parse(time.RFC3339, "2024-01-01T00:00:00Z")
	assert.Nil(t, err, fmt.Sprintf("Unexpected error parsing time: %v", err))
	return channels.Channel{
		ID:        testsutil.GenerateUUID(t),
		Name:      "channelname",
		Domain:    testsutil.GenerateUUID(t),
		Tags:      []string{"tag1", "tag2"},
		Metadata:  channels.Metadata{"key1": "value1"},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
		Status:    channels.EnabledStatus,
	}
}
