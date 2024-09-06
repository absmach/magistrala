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
	policysvc "github.com/absmach/magistrala/pkg/policy"
	policymocks "github.com/absmach/magistrala/pkg/policy/mocks"
	mgsdk "github.com/absmach/magistrala/pkg/sdk/go"
	sdkmocks "github.com/absmach/magistrala/pkg/sdk/mocks"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	streamID       = "magistrala.bootstrap"
	email          = "user@example.com"
	validToken     = "validToken"
	invalidToken   = "invalid"
	unknownThingID = "unknown"
	channelsNum    = 3
	defaultTimout  = 5

	configPrefix        = "config."
	configCreate        = configPrefix + "create"
	configView          = configPrefix + "view"
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

type testVariable struct {
	svc    bootstrap.Service
	boot   *mocks.ConfigRepository
	auth   *authmocks.AuthServiceClient
	policy *policymocks.PolicyClient
	sdk    *sdkmocks.SDK
}

func newTestVariable(t *testing.T, redisURL string) testVariable {
	boot := new(mocks.ConfigRepository)
	auth := new(authmocks.AuthServiceClient)
	policy := new(policymocks.PolicyClient)
	sdk := new(sdkmocks.SDK)
	idp := uuid.NewMock()
	svc := bootstrap.New(auth, policy, boot, sdk, encKey, idp)
	publisher, err := store.NewPublisher(context.Background(), redisURL, streamID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	svc = producer.NewEventStoreMiddleware(svc, publisher)
	return testVariable{
		svc:    svc,
		boot:   boot,
		auth:   auth,
		policy: policy,
		sdk:    sdk,
	}
}

func TestAdd(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tv := newTestVariable(t, redisURL)

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
		id           string
		domainID     string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		thingErr     error
		channel      []bootstrap.Channel
		listErr      error
		saveErr      error
		err          error
		event        map[string]interface{}
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
			desc:        "create config with invalid token",
			config:      config,
			token:       invalidToken,
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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall := tv.sdk.On("Thing", tc.config.ThingID, tc.token).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, errors.NewSDKError(tc.thingErr))
		repoCall := tv.boot.On("ListExisting", context.Background(), domainID, mock.Anything).Return(tc.config.Channels, tc.listErr)
		repoCall1 := tv.boot.On("Save", context.Background(), mock.Anything, mock.Anything).Return(mock.Anything, tc.saveErr)

		_, err := tv.svc.Add(context.Background(), tc.token, tc.config)
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
	}
}

func TestView(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tv := newTestVariable(t, redisURL)

	nonExisting := config
	nonExisting.ThingID = unknownThingID

	cases := []struct {
		desc         string
		config       bootstrap.Config
		token        string
		id           string
		domainID     string
		authResponse *magistrala.AuthorizeRes
		authorizeErr error
		identifyErr  error
		retrieveErr  error
		err          error
		event        map[string]interface{}
	}{
		{
			desc:         "view successfully",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			err:          nil,
			event: map[string]interface{}{
				"thing_id":    config.ThingID,
				"domain_id":   config.DomainID,
				"name":        config.Name,
				"channels":    config.Channels,
				"external_id": config.ExternalID,
				"content":     config.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configView,
			},
		},
		{
			desc:        "view with invalid token",
			config:      config,
			token:       invalidToken,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:         "view with failed authorization",
			config:       config,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
			event:        nil,
		},
		{
			desc:         "view with failed retrieve",
			config:       nonExisting,
			token:        validToken,
			id:           validID,
			domainID:     domainID,
			authResponse: &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:  svcerr.ErrViewEntity,
			err:          svcerr.ErrViewEntity,
			event:        nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := tv.boot.On("RetrieveByID", context.Background(), tc.domainID, tc.config.ThingID).Return(config, tc.retrieveErr)

		_, err := tv.svc.View(context.Background(), tc.token, tc.config.ThingID)
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

func TestUpdate(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tv := newTestVariable(t, redisURL)

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
	nonExisting.ThingID = unknownThingID

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
				"domain_id":   domainID,
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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := tv.boot.On("Update", context.Background(), mock.Anything).Return(tc.updateErr)
		err := tv.svc.Update(context.Background(), tc.token, tc.config)
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

	tv := newTestVariable(t, redisURL)

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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.id, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		sdkCall := tv.sdk.On("Channel", mock.Anything, tc.token).Return(mgsdk.Channel{}, tc.channelErr)
		repoCall := tv.boot.On("RetrieveByID", context.Background(), tc.domainID, tc.configID).Return(config, tc.retrieveErr)
		repoCall1 := tv.boot.On("ListExisting", context.Background(), domainID, mock.Anything, mock.Anything).Return(config.Channels, tc.listErr)
		repoCall2 := tv.boot.On("UpdateConnections", context.Background(), tc.domainID, tc.configID, mock.Anything, tc.connections).Return(tc.updateErr)
		err := tv.svc.UpdateConnections(context.Background(), tc.token, tc.configID, tc.connections)
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

	tv := newTestVariable(t, redisURL)

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
			desc:        "update cert with invalid token",
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
			desc:         "update cert with failed authorization",
			configID:     config.ThingID,
			token:        validToken,
			userID:       validID,
			domainID:     domainID,
			clientCert:   "clientCert",
			clientKey:    "clientKey",
			caCert:       "caCert",
			authResponse: &magistrala.AuthorizeRes{Authorized: false},
			authorizeErr: svcerr.ErrAuthorization,
			err:          svcerr.ErrAuthorization,
			event:        nil,
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
			desc:         "update cert with empty client certificate",
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
			desc:         "update cert with empty client key",
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
			desc:         "update cert with empty CA certificate",
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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := tv.boot.On("UpdateCert", context.Background(), tc.domainID, tc.configID, tc.clientCert, tc.clientKey, tc.caCert).Return(config, tc.updateErr)
		_, err := tv.svc.UpdateCert(context.Background(), tc.token, tc.configID, tc.clientCert, tc.clientKey, tc.caCert)

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
	tv := newTestVariable(t, redisURL)

	numThings := 101
	var c bootstrap.Config
	saved := make([]bootstrap.Config, 0)
	for i := 0; i < numThings; i++ {
		c := config
		c.ExternalID = testsutil.GenerateUUID(t)
		c.ExternalKey = testsutil.GenerateUUID(t)
		c.Name = fmt.Sprintf("%s-%d", config.Name, i)
		if i == 41 {
			c.State = bootstrap.Active
		}
		saved = append(saved, c)
	}

	cases := []struct {
		desc                string
		token               string
		userID              string
		domainID            string
		config              bootstrap.ConfigsPage
		filter              bootstrap.Filter
		offset              uint64
		limit               uint64
		identifyErr         error
		superAdminAuthRes   *magistrala.AuthorizeRes
		domainAdminAuthRes  *magistrala.AuthorizeRes
		listObjectsResponse policysvc.PolicyPage
		listObjectsErr      error
		superAdmiAuthErr    error
		domainAdmiAuthErr   error
		retrieveErr         error
		err                 error
		event               map[string]interface{}
	}{
		{
			desc:              "list successfully as super admin",
			token:             validToken,
			userID:            validID,
			domainID:          domainID,
			superAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
			event: map[string]interface{}{
				"thing_id":    c.ThingID,
				"domain_id":   c.DomainID,
				"name":        c.Name,
				"channels":    c.Channels,
				"external_id": c.ExternalID,
				"content":     c.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configList,
			},
		},
		{
			desc:               "list successfully as domain admin",
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: true},
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
			event: map[string]interface{}{
				"thing_id":    c.ThingID,
				"domain_id":   c.DomainID,
				"name":        c.Name,
				"channels":    c.Channels,
				"external_id": c.ExternalID,
				"content":     c.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configList,
			},
		},
		{
			desc:               "list successfully as non admin",
			token:              validToken,
			userID:             validID,
			domainID:           domainID,
			superAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes: &magistrala.AuthorizeRes{Authorized: false},
			config: bootstrap.ConfigsPage{
				Total:   uint64(len(saved)),
				Offset:  0,
				Limit:   10,
				Configs: saved[0:10],
			},
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			err:                 nil,
			event: map[string]interface{}{
				"thing_id":    c.ThingID,
				"domain_id":   c.DomainID,
				"name":        c.Name,
				"channels":    c.Channels,
				"external_id": c.ExternalID,
				"content":     c.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configList,
			},
		},
		{
			desc:        "list with invalid token",
			token:       invalidToken,
			filter:      bootstrap.Filter{},
			offset:      0,
			limit:       10,
			identifyErr: svcerr.ErrAuthentication,
			err:         svcerr.ErrAuthentication,
			event:       nil,
		},
		{
			desc:                "list as super admin with failed authorization",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			superAdmiAuthErr:    svcerr.ErrAuthorization,
			err:                 nil,
			event:               nil,
		},
		{
			desc:                "list as domain admin with failed authorization",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			superAdmiAuthErr:    svcerr.ErrAuthorization,
			domainAdmiAuthErr:   svcerr.ErrAuthorization,
			err:                 nil,
			event:               nil,
		},
		{
			desc:                "list as non admin with failed list all objects",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			listObjectsErr:      svcerr.ErrNotFound,
			err:                 svcerr.ErrNotFound,
			event:               nil,
		},

		{
			desc:                "list as super admin with failed retrieve all",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:         nil,
			err:                 nil,
			event:               nil,
		},
		{
			desc:                "list as domain admin with failed retrieve all",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: true},
			retrieveErr:         nil,
			err:                 nil,
			event:               nil,
		},
		{
			desc:                "list as non admin with failed retrieve all",
			token:               validToken,
			userID:              validID,
			domainID:            domainID,
			filter:              bootstrap.Filter{},
			offset:              0,
			limit:               10,
			listObjectsResponse: policysvc.PolicyPage{},
			superAdminAuthRes:   &magistrala.AuthorizeRes{Authorized: false},
			domainAdminAuthRes:  &magistrala.AuthorizeRes{Authorized: false},
			retrieveErr:         nil,
			err:                 nil,
			event:               nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.userID,
			Permission:  policysvc.AdminPermission,
			ObjectType:  policysvc.PlatformType,
			Object:      policysvc.MagistralaObject,
		}).Return(tc.superAdminAuthRes, tc.superAdmiAuthErr)
		authCall2 := tv.auth.On("Authorize", context.Background(), &magistrala.AuthorizeReq{
			SubjectType: policysvc.UserType,
			SubjectKind: policysvc.UsersKind,
			Subject:     tc.userID,
			Permission:  policysvc.AdminPermission,
			ObjectType:  policysvc.DomainType,
			Object:      tc.domainID,
		}).Return(tc.domainAdminAuthRes, tc.domainAdmiAuthErr)
		authCall3 := tv.policy.On("ListAllObjects", mock.Anything, policysvc.PolicyReq{
			SubjectType: policysvc.UserType,
			Subject:     tc.userID,
			Permission:  policysvc.ViewPermission,
			ObjectType:  policysvc.ThingType,
		}).Return(tc.listObjectsResponse, tc.listObjectsErr)
		repoCall := tv.boot.On("RetrieveAll", context.Background(), mock.Anything, mock.Anything, tc.filter, tc.offset, tc.limit).Return(tc.config, tc.retrieveErr)

		_, err := tv.svc.List(context.Background(), tc.token, tc.filter, tc.offset, tc.limit)
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
		authCall2.Unset()
		authCall3.Unset()
		repoCall.Unset()
	}
}

func TestRemove(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tv := newTestVariable(t, redisURL)

	nonExisting := config
	nonExisting.ThingID = unknownThingID

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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		authCall1 := tv.auth.On("Authorize", context.Background(), mock.Anything).Return(tc.authResponse, tc.authorizeErr)
		repoCall := tv.boot.On("Remove", context.Background(), mock.Anything, mock.Anything).Return(tc.removeErr)
		err := tv.svc.Remove(context.Background(), tc.token, tc.configID)
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

	tv := newTestVariable(t, redisURL)

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
		repoCall := tv.boot.On("RetrieveByExternalID", context.Background(), mock.Anything).Return(config, tc.retrieveErr)
		_, err = tv.svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, false)
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

	tv := newTestVariable(t, redisURL)

	cases := []struct {
		desc         string
		id           string
		userID       string
		domainID     string
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
		authCall := tv.auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: tc.userID, DomainId: tc.domainID}, tc.identifyErr)
		repoCall := tv.boot.On("RetrieveByID", context.Background(), tc.domainID, tc.id).Return(config, tc.retrieveErr)
		sdkCall1 := tv.sdk.On("Connect", mock.Anything, mock.Anything).Return(errors.NewSDKError(tc.connectErr))
		repoCall1 := tv.boot.On("ChangeState", context.Background(), mock.Anything, mock.Anything, mock.Anything).Return(tc.stateErr)
		err := tv.svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
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
		sdkCall1.Unset()
		repoCall.Unset()
		repoCall1.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	tv := newTestVariable(t, redisURL)

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
		repoCall := tv.boot.On("UpdateChannel", context.Background(), mock.Anything).Return(tc.err)
		err := tv.svc.UpdateChannelHandler(context.Background(), tc.channel)
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

	tv := newTestVariable(t, redisURL)

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
		repoCall := tv.boot.On("RemoveChannel", context.Background(), mock.Anything).Return(tc.err)
		err := tv.svc.RemoveChannelHandler(context.Background(), tc.channelID)
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

	tv := newTestVariable(t, redisURL)

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
		repoCall := tv.boot.On("RemoveThing", context.Background(), mock.Anything).Return(tc.err)
		err := tv.svc.RemoveConfigHandler(context.Background(), tc.configID)
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

	tv := newTestVariable(t, redisURL)

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
			desc:      "connect non-existing thing handler",
			channelID: channel.ID,
			thingID:   "unknown",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "connect thing handler with empty thing ID",
			channelID: channel.ID,
			thingID:   "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "connect thing handler with empty channel ID",
			channelID: "",
			thingID:   "1",
			err:       nil,
			event:     nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := tv.boot.On("ConnectThing", context.Background(), mock.Anything, mock.Anything).Return(tc.err)
		err := tv.svc.ConnectThingHandler(context.Background(), tc.channelID, tc.thingID)
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

	tv := newTestVariable(t, redisURL)

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
			desc:      "remove non-existing thing handler",
			channelID: "unknown",
			err:       nil,
		},
		{
			desc:      "remove thing handler with empty thing ID",
			channelID: channel.ID,
			thingID:   "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "remove thing handler with empty channel ID",
			channelID: "",
			err:       nil,
			event:     nil,
		},
		{
			desc:      "remove thing handler successfully",
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
		repoCall := tv.boot.On("DisconnectThing", context.Background(), tc.channelID, tc.thingID).Return(tc.err)
		err := tv.svc.DisconnectThingHandler(context.Background(), tc.channelID, tc.thingID)
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
