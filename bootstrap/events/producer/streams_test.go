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
	validToken    = "validToken"
	invalidToken  = "invalid"
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
	instanceID = "5de9b29a-feb9-11ed-be56-0242ac120002"
)

var (
	encKey = []byte("1234567891011121")

	domainID = testsutil.GenerateUUID(&testing.T{})
	validID  = testsutil.GenerateUUID(&testing.T{})

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
		desc               string
		config             bootstrap.Config
		token              string
		id                 string
		domainID           string
		authResponse       *magistrala.AuthorizeRes
		authorizeErr       error
		identifyErr        error
		thingErr           error
		channelsByThingErr error
		channel            []bootstrap.Channel
		listErr            error
		saveErr            error
		err                error
		event              map[string]interface{}
	}{
		{
			desc:         "create config successfully",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			channel:      config.Channels,
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
			desc:        "create config with empty token",
			config:      config,
			token:       "",
			event:       nil,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
		},
		{
			desc:         "create config with failed authorization",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			event:        nil,
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
		},
		{
			desc:         "create config with failed to fetch thing",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event:        nil,
			thingErr:     svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
		{
			desc:               "create config with failed to fetch channels",
			config:             config,
			token:              validToken,
			id:                 validID,
			domainID:           domainID,
			authResponse:       &magistrala.AuthorizeRes{Authorized: true},
			channel:            invalidConfig.Channels,
			event:              nil,
			channelsByThingErr: svcerr.ErrNotFound,
			err:                svcerr.ErrNotFound,
		},
		{
			desc:         "create config with failed to list existing",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event:        nil,
			listErr:      svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
		},
		{
			desc:         "create invalid config",
			config:       invalidConfig,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event:        nil,
			listErr:      svcerr.ErrMalformedEntity,
			err:          svcerr.ErrMalformedEntity,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall := sdk.On("Thing", tc.config.ThingID, tc.token).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, errors.NewSDKError(tc.thingErr))
		sdkCall1 := sdk.On("ChannelsByThing", tc.config.ThingID, mock.Anything, tc.token).Return(mgsdk.ChannelsPage{}, errors.NewSDKError(tc.channelsByThingErr))
		repoCall := boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(tc.config.Channels, tc.listErr)
		repoCall1 := boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)

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
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestView(t *testing.T) {
	svc, boot, auth, _ := newService(t, redisURL)

	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall := boot.On("RetrieveByID", context.Background(), domainID, config.ThingID).Return(config, nil)
	svcConfig, svcErr := svc.View(context.Background(), validToken, config.ThingID)
	authCall.Unset()
	authCall1.Unset()
	repoCall.Unset()

	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall = boot.On("RetrieveByID", context.Background(), domainID, config.ThingID).Return(config, nil)
	esConfig, esErr := svc.View(context.Background(), validToken, config.ThingID)
	authCall.Unset()
	authCall1.Unset()
	repoCall.Unset()

	assert.Equal(t, svcConfig, esConfig, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfig, esConfig))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestUpdate(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, _ := newService(t, redisURL)

	c := config

	ch1 := channel
	ch1.ID = testsutil.GenerateUUID(t)

	ch2 := channel
	ch2.ID = testsutil.GenerateUUID(t)

	c.Channels = append(c.Channels, ch1, ch2)

	modified := c
	modified.Content = "new-config"
	modified.Name = "new name"

	nonExisting := config
	nonExisting.ThingID = "unknown"

	channels := []string{modified.Channels[0].ID, modified.Channels[1].ID}

	cases := []struct {
		desc         string
		config       bootstrap.Config
		token        string
		id           string
		domainID     string
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
			id:           validID,
			domainID:     domainID,
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
			desc:        "update with invalid token",
			config:      modified,
			token:       invalidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "update with failed authorization",
			config:       modified,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
			event:        nil,
		},
		{
			desc:         "update with failed update",
			config:       nonExisting,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := boot.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
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
		repoCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	cases := []struct {
		desc         string
		configID     string
		id           string
		domainID     string
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
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  config.ThingID,
				"channels":  "2",
				"timestamp": time.Now().Unix(),
				"operation": thingUpdateConnections,
			},
		},
		{
			desc:        "update connections with invalid token",
			configID:    config.ThingID,
			token:       invalidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "update connections with failed authorization",
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
			event:        nil,
		},
		{
			desc:         "update connections with failed channel fetch",
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{"256"},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			channelErr:   errors.NewSDKError(svcerr.ErrNotFound),
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
		{
			desc:         "update connections with failed RetrieveByID",
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:  svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
		{
			desc:         "update connections with failed ListExisting",
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			listErr:      svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
		{
			desc:         "update connections with failed UpdateConnections",
			configID:     config.ThingID,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			connections:  []string{config.Channels[0].ID},
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			updateErr:    svcerr.ErrUpdateEntity,
			err:          svcerr.ErrUpdateEntity,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall := sdk.On("Channel", mock.Anything, tc.token).Return(mgsdk.Channel{}, tc.channelErr)
		repoCall := boot.On("RetrieveByID", context.Background(), mock.Anything, mock.Anything).Return(config, tc.retrieveErr)
		repoCall1 := boot.On("ListExisting", context.Background(), domainID, mock.Anything, mock.Anything).Return(config.Channels, tc.listErr)
		repoCall2 := boot.On("UpdateConnections", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(tc.updateErr)
		err := svc.UpdateConnections(context.Background(), tc.token, tc.configID, tc.connections)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, _ := newService(t, redisURL)

	cases := []struct {
		desc         string
		configID     string
		userID       string
		domainID     string
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
			configID:     config.ThingID,
			userID:       validID,
			domainID:     domainID,
			token:        validToken,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_key":   config.ThingKey,
				"client_cert": "clientCert",
				"client_key":  "clientKey",
				"ca_cert":     "caCert",
				"operation":   certUpdate,
			},
		},
		{
			desc:        "invalid token",
			configID:    config.ThingID,
			token:       "invalid",
			clientCert:  "clientCert",
			clientKey:   "clientKey",
			caCert:      "caCert",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "update cert with failed update",
			configID:     "invalidThingID",
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
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
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			clientCert:   "",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:         "empty client key",
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			clientCert:   "clientCert",
			clientKey:    "",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:         "empty CA certificate",
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "",
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event:        nil,
		},
		{
			desc:         "successful update without CA certificate",
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "",
			err:          nil,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			event: map[string]interface{}{
				"thing_key":   config.ThingKey,
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
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := boot.On("UpdateCert", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(config, tc.updateErr)
		_, err := svc.UpdateCert(context.Background(), tc.token, tc.configID, tc.clientCert, tc.clientKey, tc.caCert)
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
		repoCall.Unset()
	}
}

func TestList(t *testing.T) {
	svc, boot, auth, _ := newService(t, redisURL)

	offset := uint64(0)
	limit := uint64(10)
	authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall := boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bootstrap.ConfigsPage{})
	svcConfigs, svcErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	authCall.Unset()
	authCall1.Unset()
	repoCall.Unset()

	authCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID, DomainId: domainID}, nil)
	authCall1 = auth.On("Authorize", context.Background(), mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
	repoCall = boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(bootstrap.ConfigsPage{})
	esConfigs, esErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	authCall.Unset()
	authCall1.Unset()
	repoCall.Unset()
	assert.Equal(t, svcConfigs, esConfigs)
	assert.Equal(t, svcErr, esErr)
}

func TestRemove(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, _ := newService(t, redisURL)

	nonExisting := config
	nonExisting.ThingID = "unknown"

	cases := []struct {
		desc         string
		configID     string
		userID       string
		domainID     string
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
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  config.ThingID,
				"timestamp": time.Now().Unix(),
				"operation": configRemove,
			},
		},
		{
			desc:        "remove config with invalid credentials",
			configID:    config.ThingID,
			token:       "",
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "remove config with failed authorization",
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
			event:        nil,
		},
		{
			desc:         "remove config with failed removal",
			configID:     nonExisting.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			removeErr:    svcerr.ErrNotFound,
			err:          svcerr.ErrNotFound,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := boot.On("Remove", context.Background(), mock.Anything, mock.Anything).Return(tc.removeErr)
		err := svc.Remove(context.Background(), tc.token, tc.configID)
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
		repoCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

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
			externalID:  config.ExternalID,
			externalKey: config.ExternalKey,
			err:         nil,
			event: map[string]interface{}{
				"external_id": config.ExternalID,
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
		repoCall := boot.On("RetrieveByExternalID", context.Background(), mock.Anything).Return(config, tc.retrieveErr)
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
		repoCall.Unset()
	}
}

func TestChangeState(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, auth, sdk := newService(t, redisURL)

	cases := []struct {
		desc               string
		id                 string
		userID             string
		domainID           string
		token              string
		state              bootstrap.State
		authResponse       *magistrala.AuthorizeRes
		channelsByThingErr error
		authorizeErr       error
		connectErr         error
		retrieveErr        error
		stateErr           error
		identifyErr        error
		err                error
		event              map[string]interface{}
	}{
		{
			desc:         "change state to active",
			id:           config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			state:        bootstrap.Active,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":  config.ThingID,
				"state":     bootstrap.Active.String(),
				"timestamp": time.Now().Unix(),
				"operation": thingStateChange,
			},
		},
		{
			desc:        "change state invalid credentials",
			id:          config.ThingID,
			token:       "invalid",
			state:       bootstrap.Inactive,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:        "change state with failed retrieve by ID",
			id:          "",
			token:       validToken,
			userID:      validID,
			domainID:    domainID,
			state:       bootstrap.Active,
			retrieveErr: svcerr.ErrNotFound,
			err:         svcerr.ErrNotFound,
			event:       nil,
		},
		{
			desc:               "change state with failed channels by thing check",
			id:                 config.ThingID,
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			state:              bootstrap.Active,
			channelsByThingErr: svcerr.ErrNotFound,
			err:                svcerr.ErrNotFound,
			event:              nil,
		},
		{
			desc:       "change state with failed connect",
			id:         config.ThingID,
			token:      validToken,
			userID:     validID,
			domainID:   domainID,
			state:      bootstrap.Active,
			connectErr: bootstrap.ErrThings,
			err:        bootstrap.ErrThings,
			event:      nil,
		},
		{
			desc:     "change state unsuccessfully",
			id:       config.ThingID,
			token:    validToken,
			userID:   validID,
			domainID: domainID,
			state:    bootstrap.Active,
			stateErr: svcerr.ErrUpdateEntity,
			err:      svcerr.ErrUpdateEntity,
			event:    nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		repoCall := boot.On("RetrieveByID", context.Background(), mock.Anything, mock.Anything).Return(config, tc.retrieveErr)
		sdkCall := sdk.On("ChannelsByThing", tc.id, mock.Anything, tc.token).Return(mgsdk.ChannelsPage{}, errors.NewSDKError(tc.channelsByThingErr))
		sdkCall1 := sdk.On("Connect", mock.Anything, mock.Anything).Return(errors.NewSDKError(tc.connectErr))
		repoCall1 := boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
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
		sdkCall.Unset()
		sdkCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

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
		repoCall := boot.On("UpdateChannel", context.Background(), mock.Anything).Return(tc.err)
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
		repoCall.Unset()
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

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
		repoCall := boot.On("RemoveChannel", context.Background(), mock.Anything).Return(tc.err)
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
		repoCall.Unset()
	}
}

func TestRemoveConfigHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, boot, _, _ := newService(t, redisURL)

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
		repoCall := boot.On("RemoveThing", context.Background(), mock.Anything).Return(tc.err)
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
		repoCall.Unset()
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
