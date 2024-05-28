// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/absmach/magistrala"
	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/bootstrap/events/producer"
	"github.com/absmach/magistrala/bootstrap/mocks"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/events/store"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	streamID      = "magistrala.bootstrap"
	email         = "user@example.com"
	validToken    = "token"
	channelsNum   = 3
	defaultTimout = 5

	configPrefix        = "config."
	configCreate        = configPrefix + "create"
	configUpdate        = configPrefix + "update"
	configRemove        = configPrefix + "remove"
	configList          = configPrefix + "list"
	configHandlerRemove = configPrefix + "remove_handler"

	thingPrefix            = "thing."
	thingBootstrap         = thingPrefix + "bootstrap"
	thingStateChange       = thingPrefix + "change_state"
	thingUpdateConnections = thingPrefix + "update_connections"
	thingConnect           = thingPrefix + "connect"
	thingDisconnect        = thingPrefix + "disconnect"

	channelPrefix        = "group."
	channelHandlerRemove = channelPrefix + "remove_handler"
	channelUpdateHandler = channelPrefix + "update_handler"

	certUpdate = "cert.update"
	validID    = "d4ebb847-5d0e-4e46-bdd9-b6aceaaa3a22"
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
)

var (
	encKey = []byte("1234567891011121")

	domainID = testsutil.GenerateUUID(&testing.T{})

	channel = bootstrap.Channel{
		ID:       testsutil.GenerateUUID(&testing.T{}),
		Name:     "name",
		Metadata: map[string]interface{}{"name": "value"},
	}

	config = bootstrap.Config{
		ThingID:     testsutil.GenerateUUID(&testing.T{}),
		ThingKey:    testsutil.GenerateUUID(&testing.T{}),
		ExternalID:  testsutil.GenerateUUID(&testing.T{}),
		ExternalKey: testsutil.GenerateUUID(&testing.T{}),
		Channels:    []bootstrap.Channel{channel},
		Content:     "config",
	}
)

func newService(t *testing.T, url string) (bootstrap.Service, *mocks.ConfigRepository, *authmocks.AuthClient, *sdkmocks.SDK) {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, boot, sdk, encKey, idp)
	publisher, err := store.NewPublisher(context.Background(), url, streamID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	svc = producer.NewEventStoreMiddleware(svc, publisher)

	return svc, boot, auth, sdk
}

func TestAdd(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	var channels []string
	for _, ch := range config.Channels {
		channels = append(channels, ch.ID)
	}

	invalidConfig := config
	invalidConfig.Channels = []bootstrap.Channel{{ID: "empty"}}
	invalidConfig.Channels = []bootstrap.Channel{{ID: "empty"}}

	cases := []struct {
		desc         string
		config       bootstrap.Config
		token        string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		channelErr   error
		listErr      error
		saveErr      error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "create config successfully",
			config:       config,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event: map[string]interface{}{
				"thing_id":    "1",
				"domain_id":   domainID,
				"name":        config.Name,
				"channels":    channels,
				"external_id": config.ExternalID,
				"content":     config.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configCreate,
			},
			err: nil,
		},
		{
			desc:         "create invalid config",
			config:       invalidConfig,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event:        nil,
			listErr:      svcerr.ErrMalformedEntity,
			err:          svcerr.ErrMalformedEntity,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall := sdk.On("Thing", tc.config.ThingID, tc.token).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, errors.NewSDKError(tc.authorizeErr))
		sdkCall1 := sdk.On("Channel", channel.ID, tc.token).Return(toChannel(tc.config.Channels[0]), errors.NewSDKError(tc.channelErr))
		svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(tc.config.Channels, tc.listErr)
		svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)

		_, err := svc.Add(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}

		test(t, tc.event, event, tc.desc)

		authCall.Unset()
		authCall1.Unset()
		sdkCall.Unset()
		sdkCall1.Unset()
		svcCall.Unset()
		svcCall1.Unset()
	}
}

func TestView(t *testing.T) {
	svc, boot, auth, sdk := newService(t, redisURL)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", config.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", channel.ID, validID).Return(toChannel(config.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	sdkCall = sdk.On("Thing", saved.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, errors.NewSDKError(err))
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	svcCall = boot.On("RetrieveByID", context.Background(), domainID, config.ThingID).Return(config, nil)
	svcConfig, svcErr := svc.View(context.Background(), validToken, saved.ThingID)
	authCall.Unset()
	sdkCall.Unset()
	authCall1.Unset()
	svcCall.Unset()

	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	sdkCall = sdk.On("Thing", saved.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, errors.NewSDKError(err))
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	svcCall = boot.On("RetrieveByID", context.Background(), domainID, config.ThingID).Return(config, nil)
	esConfig, esErr := svc.View(context.Background(), validToken, saved.ThingID)
	authCall.Unset()
	sdkCall.Unset()
	authCall1.Unset()
	svcCall.Unset()

	assert.Equal(t, svcConfig, esConfig, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfig, esConfig))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestUpdate(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	c := config

	ch := channel
	ch.ID = testsutil.GenerateUUID(t)

	c.Channels = append(c.Channels, ch)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", c.ThingID, validToken).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", ch.ID, validToken).Return(toChannel(c.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(c.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)

	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	modified := saved
	modified.Content = "new-config"
	modified.Name = "new name"

	nonExisting := config
	nonExisting.ThingID = "unknown"

	channels := []string{modified.Channels[0].ID, modified.Channels[1].ID}

	cases := []struct {
		desc         string
		config       bootstrap.Config
		token        string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		updateErr    error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "update config successfully",
			config:       modified,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"name":        modified.Name,
				"content":     modified.Content,
				"timestamp":   time.Now().UnixNano(),
				"operation":   configUpdate,
				"channels":    channels,
				"external_id": modified.ExternalID,
				"thing_id":    modified.ThingID,
				"owner":       validID,
				"state":       "0",
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:         "update non-existing config",
			config:       nonExisting,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		svcCall := boot.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
		err := svc.Update(context.Background(), tc.token, tc.config)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		authCall.Unset()
		authCall1.Unset()
		svcCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall1 := sdk.On("Thing", config.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall2 := sdk.On("Channel", channel.ID, validToken).Return(toChannel(config.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)

	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall1.Unset()
	sdkCall2.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		id           string
		token        string
		connections  []string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		thingErr     error
		channelErr   error
		retrieveErr  error
		listErr      error
		updateErr    error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "update connections successfully",
			id:           saved.ThingID,
			token:        validToken,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"channels":  []string{"2"},
				"timestamp": time.Now().Unix(),
				"operation": thingUpdateConnections,
			},
		},
		{
			desc:         "update connections unsuccessfully",
			id:           saved.ThingID,
			token:        validToken,
			connections:  []string{"256"},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			channelErr:   errors.NewSDKError(svcerr.ErrNotFound),
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall1 := sdk.On("Thing", tc.id, tc.token).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, tc.thingErr)
		sdkCall2 := sdk.On("Channel", mock.Anything, tc.token).Return(mgsdk.Channel{}, tc.channelErr)
		svcCall := boot.On("RetrieveByID", context.Background(), mock.Anything, mock.Anything).Return(config, tc.retrieveErr)
		svcCall1 := boot.On("ListExisting", context.Background(), domainID, mock.Anything, mock.Anything).Return(config.Channels, tc.listErr)
		svcCall2 := boot.On("UpdateConnections", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.updateErr)
		err := svc.UpdateConnections(context.Background(), tc.token, tc.id, tc.connections)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}

		test(t, tc.event, event, tc.desc)
		authCall.Unset()
		authCall1.Unset()
		sdkCall1.Unset()
		sdkCall2.Unset()
		svcCall.Unset()
		svcCall1.Unset()
		svcCall2.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", config.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", channel.ID, validToken).Return(toChannel(config.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		id           string
		token        string
		clientCert   string
		clientKey    string
		caCert       string
		authResponse *magistrala.AuthorizeRes
		identifyErr  error
		authorizeErr error
		updateErr    error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "update cert successfully",
			id:           saved.ThingID,
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_key":   saved.ThingKey,
				"client_cert": "clientCert",
				"client_key":  "clientKey",
				"ca_cert":     "caCert",
				"operation":   certUpdate,
			},
		},
		{
			desc:        "invalid token",
			id:          saved.ThingID,
			token:       "invalid",
			clientCert:  "clientCert",
			clientKey:   "clientKey",
			caCert:      "caCert",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "invalid thing ID",
			id:           "invalidThingID",
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
		{
			desc:         "empty client certificate",
			id:           saved.ThingID,
			token:        validToken,
			clientCert:   "",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:         "empty client key",
			id:           saved.ThingID,
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:         "empty CA certificate",
			id:           saved.ThingID,
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:        "update cert with invalid token",
			id:          saved.ThingID,
			token:       "invalid",
			clientCert:  "clientCert",
			clientKey:   "clientKey",
			caCert:      "caCert",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "successful update without CA certificate",
			id:           saved.ThingID,
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "",
			err:          nil,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event: map[string]interface{}{
				"thing_key":   saved.ThingKey,
				"client_cert": "clientCert",
				"client_key":  "clientKey",
				"ca_cert":     "caCert",
				"operation":   certUpdate,
				"timestamp":   time.Now().Unix(),
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		svcCall := boot.On("UpdateCert", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(config, tc.updateErr)
		_, err := svc.UpdateCert(context.Background(), tc.token, tc.id, tc.clientCert, tc.clientKey, tc.caCert)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}

		test(t, tc.event, event, tc.desc)

		authCall.Unset()
		authCall1.Unset()
		svcCall.Unset()
	}
}

func TestList(t *testing.T) {
	svc, boot, auth, sdk := newService(t, redisURL)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", config.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", channel.ID, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	_, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	offset := uint64(0)
	limit := uint64(10)
	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	svcCall = boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bootstrap.ConfigsPage{})
	svcConfigs, svcErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	authCall.Unset()
	authCall1.Unset()
	svcCall.Unset()

	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	svcCall = boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bootstrap.ConfigsPage{})
	esConfigs, esErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	authCall.Unset()
	authCall1.Unset()
	svcCall.Unset()
	assert.Equal(t, svcConfigs, esConfigs)
	assert.Equal(t, svcErr, esErr)
}

func TestRemove(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	c := config

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", c.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall2 := sdk.On("Channel", channel.ID, validToken).Return(toChannel(config.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall2.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		id           string
		token        string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		removeErr    error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "remove config successfully",
			id:           saved.ThingID,
			token:        validToken,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"timestamp": time.Now().Unix(),
				"operation": configRemove,
			},
		},
		{
			desc:        "remove config with invalid credentials",
			id:          saved.ThingID,
			token:       "",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		svcCall := boot.On("Remove", context.Background(), mock.Anything, mock.Anything).Return(tc.removeErr)
		err := svc.Remove(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}

		test(t, tc.event, event, tc.desc)
		authCall.Unset()
		authCall1.Unset()
		svcCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	c := config

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", c.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", channel.ID, validToken).Return(mgsdk.Channel{}, nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		err         error
		retrieveErr error
		event       map[string]interface{}
	}{
		{
			desc:        "bootstrap successfully",
			externalID:  saved.ExternalID,
			externalKey: saved.ExternalKey,
			err:         nil,
			event: map[string]interface{}{
				"external_id": saved.ExternalID,
				"success":     "1",
				"timestamp":   time.Now().Unix(),
				"operation":   thingBootstrap,
			},
		},
		{
			desc:        "bootstrap with an error",
			externalID:  "external_id1",
			externalKey: "external_id",
			retrieveErr: bootstrap.ErrBootstrap,
			err:         bootstrap.ErrBootstrap,
			event: map[string]interface{}{
				"external_id": "external_id",
				"success":     "0",
				"timestamp":   time.Now().Unix(),
				"operation":   thingBootstrap,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		svcCall := boot.On("RetrieveByExternalID", context.Background(), mock.Anything).Return(config, tc.retrieveErr)
		_, err = svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, false)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}
		test(t, tc.event, event, tc.desc)
		svcCall.Unset()
	}
}

func TestChangeState(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	c := config

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	sdkCall := sdk.On("Thing", c.ThingID, validToken).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	sdkCall1 := sdk.On("Channel", channel.ID, validToken).Return(toChannel(c.Channels[0]), nil)
	svcCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(config.Channels, nil)
	svcCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	authCall.Unset()
	authCall1.Unset()
	sdkCall.Unset()
	sdkCall1.Unset()
	svcCall.Unset()
	svcCall1.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc         string
		id           string
		token        string
		state        bootstrap.State
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		connectErr   error
		retrieveErr  error
		stateErr     error
		identifyErr  error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "change state to active",
			id:           saved.ThingID,
			token:        validToken,
			state:        bootstrap.Active,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"state":     bootstrap.Active.String(),
				"timestamp": time.Now().Unix(),
				"operation": thingStateChange,
			},
		},
		{
			desc:        "change state invalid credentials",
			id:          saved.ThingID,
			token:       "invalid",
			state:       bootstrap.Inactive,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall = sdk.On("Connect", mock.Anything, mock.Anything).Return(tc.connectErr)
		svcCall := boot.On("RetrieveByID", context.Background(), mock.Anything, mock.Anything).Return(config, tc.retrieveErr)
		svcCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
		err := svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			event := streams[0].Messages
			lastID = event[0].ID
		}

		test(t, tc.event, event, tc.desc)
		authCall.Unset()
		authCall1.Unset()
		sdkCall.Unset()
		svcCall.Unset()
		svcCall1.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel bootstrap.Channel
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "update channel handler successfully",
			channel: channel,
			err:     nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"metadata":    "{\"name\":\"value\"}",
				"name":        channel.Name,
				"operation":   channelUpdateHandler,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:    "update non-existing channel handler",
			channel: bootstrap.Channel{ID: "unknown", Name: "NonExistingChannel"},
			err:     nil,
			event:   nil,
		},
		{
			desc:    "update channel handler with empty ID",
			channel: bootstrap.Channel{Name: "ChannelWithEmptyID"},
			err:     nil,
			event:   nil,
		},
		{
			desc:    "update channel handler with empty name",
			channel: bootstrap.Channel{ID: "3"},
			err:     nil,
			event:   nil,
		},
		{
			desc:    "update channel handler successfully with modified fields",
			channel: channel,
			err:     nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"metadata":    "{\"name\":\"value\"}",
				"name":        channel.Name,
				"operation":   channelUpdateHandler,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		svcCall := boot.On("UpdateChannel", context.Background(), mock.Anything).Return(tc.err)
		err := svc.UpdateChannelHandler(context.Background(), tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}
		test(t, tc.event, event, tc.desc)
		svcCall.Unset()
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		err       error
		event     map[string]interface{}
	}{
		{
			desc:      "remove channel handler successfully",
			channelID: channel.ID,
			err:       nil,
			event: map[string]interface{}{
				"config_id":   channel.ID,
				"operation":   channelHandlerRemove,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:      "remove non-existing channel handler",
			channelID: "unknown",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "remove channel handler with empty ID",
			channelID: "",
			err:       nil,
			event:     nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		svcCall := boot.On("RemoveChannel", context.Background(), mock.Anything).Return(tc.err)
		err := svc.RemoveChannelHandler(context.Background(), tc.channelID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		svcCall.Unset()
	}
}

func TestRemoveConfigHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc     string
		configID string
		err      error
		event    map[string]interface{}
	}{
		{
			desc:     "remove config handler successfully",
			configID: channel.ID,
			err:      nil,
			event: map[string]interface{}{
				"config_id":   channel.ID,
				"operation":   configHandlerRemove,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:     "remove non-existing config handler",
			configID: "unknown",
			err:      nil,
			event:    nil,
		},
		{
			desc:     "remove config handler with empty ID",
			configID: "",
			err:      nil,
			event:    nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		svcCall := boot.On("RemoveThing", context.Background(), mock.Anything).Return(tc.err)
		err := svc.RemoveConfigHandler(context.Background(), tc.configID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		svcCall.Unset()
	}
}

func TestConnectThingHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		thingID   string
		err       error
		event     map[string]interface{}
	}{
		{
			desc:      "connect thing handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingConnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:      "add non-existing channel handler",
			channelID: "unknown",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "add channel handler with empty ID",
			channelID: "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "add channel handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingConnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := boot.On("ConnectThing", context.Background(), tc.channelID, tc.thingID).Return(tc.err)
		err := svc.ConnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		repoCall.Unset()
	}
}

func TestConnectThingHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		thingID   string
		err       error
		event     map[string]interface{}
	}{
		{
			desc:      "connect thing handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingConnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:      "add non-existing channel handler",
			channelID: "unknown",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "add channel handler with empty ID",
			channelID: "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "add channel handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingConnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := boot.On("ConnectThing", context.Background(), tc.channelID, tc.thingID).Return(tc.err)
		err := svc.ConnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		repoCall.Unset()
	}
}

func TestDisconnectThingHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc      string
		channelID string
		thingID   string
		err       error
		event     map[string]interface{}
	}{
		{
			desc:      "disconnect thing handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingDisconnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:      "remove non-existing channel handler",
			channelID: "unknown",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "remove channel handler with empty ID",
			channelID: "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "remove channel handler successfully",
			channelID: channel.ID,
			thingID:   "1",
			err:       nil,
			event: map[string]interface{}{
				"channel_id":  channel.ID,
				"thing_id":    "1",
				"operation":   thingDisconnect,
				"timestamp":   time.Now().UnixNano(),
				"occurred_at": time.Now().UnixNano(),
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := boot.On("DisconnectThing", context.Background(), tc.channelID, tc.thingID).Return(tc.err)
		err := svc.DisconnectThingHandler(context.Background(), tc.channelID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(context.Background(), &redis.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			event["timestamp"] = msg.ID
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
		svcCall.Unset()
	}
}

func test(t *testing.T, expected, actual map[string]interface{}, description string) {
	if expected != nil && actual != nil {
		ts1 := expected["timestamp"].(int64)
		ats := actual["timestamp"].(string)
		ts2, err := strconv.ParseInt(strings.Split(ats, "-")[0], 10, 64)
		require.Nil(t, err, fmt.Sprintf("%s: expected to get a valid timestamp, got %s", description, err))
		ts1 = ts1 / 1e9
		ts2 = ts2 / 1e3
		if assert.WithinDuration(t, time.Unix(ts1, 0), time.Unix(ts2, 0), time.Second, fmt.Sprintf("%s: timestamp is not in valid range of 1 second", description)) {
			delete(expected, "timestamp")
			delete(actual, "timestamp")
		}

		oa1 := expected["occurred_at"].(int64)
		aoa := actual["occurred_at"].(string)
		oa2, err := strconv.ParseInt(aoa, 10, 64)
		require.Nil(t, err, fmt.Sprintf("%s: expected to get a valid occurred_at, got %s", description, err))
		oa1 = oa1 / 1e9
		oa2 = oa2 / 1e9
		if assert.WithinDuration(t, time.Unix(oa1, 0), time.Unix(oa2, 0), time.Second, fmt.Sprintf("%s: occurred_at is not in valid range of 1 second", description)) {
			delete(expected, "occurred_at")
			delete(actual, "occurred_at")
		}

		exchs := expected["channels"].([]interface{})
		achs := actual["channels"].([]interface{})

		if exchs != nil && achs != nil {
			if assert.Len(t, exchs, len(achs), fmt.Sprintf("%s: got incorrect number of channels\n", description)) {
				for _, exch := range exchs {
					assert.Contains(t, achs, exch, fmt.Sprintf("%s: got incorrect channel\n", description))
				}
			}
		}

		assert.Equal(t, expected, actual, fmt.Sprintf("%s: got incorrect event\n", description))
	}
}

func toChannel(ch bootstrap.Channel) mgsdk.Channel {
	return mgsdk.Channel{
		ID:       ch.ID,
		Name:     ch.Name,
		Metadata: ch.Metadata,
	}
}
