// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package producer_test

import (
	"context"
	"encoding/json"
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

func newService(t *testing.T, url string) (bootstrap.Service, *authmocks.Service, *sdkmocks.SDK) {
	things := mocks.NewConfigsRepository()
	auth := new(authmocks.Service)
	sdk := new(sdkmocks.SDK)

	svc := bootstrap.New(auth, things, sdk, encKey)
	publisher, err := store.NewPublisher(context.Background(), url, streamID)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	svc = producer.NewEventStoreMiddleware(svc, publisher)

	return svc, auth, sdk
}

func TestAdd(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	var channels []string
	for _, ch := range config.Channels {
		channels = append(channels, ch.ID)
	}

	invalidConfig := config
	invalidConfig.Channels = []bootstrap.Channel{{ID: "empty"}}
	invalidConfig.Channels = []bootstrap.Channel{{ID: "empty"}}

	cases := []struct {
		desc   string
		config bootstrap.Config
		token  string
		err    error
		event  map[string]interface{}
	}{
		{
			desc:   "create config successfully",
			config: config,
			token:  validToken,
			err:    nil,
			event: map[string]interface{}{
				"thing_id":    "1",
				"owner":       email,
				"name":        config.Name,
				"channels":    strings.Join(channels, ", "),
				"external_id": config.ExternalID,
				"content":     config.Content,
				"timestamp":   time.Now().Unix(),
				"operation":   configCreate,
			},
		},
		{
			desc:   "create invalid config",
			config: invalidConfig,
			token:  validToken,
			err:    svcerr.ErrMalformedEntity,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: tc.config.ThingID, Credentials: mgsdk.Credentials{Secret: tc.config.ThingKey}}, errors.NewSDKError(tc.err))
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(tc.config.Channels[0]), errors.NewSDKError(tc.err))
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

		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestView(t *testing.T) {
	svc, auth, sdk := newService(t, redisURL)

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	svcConfig, svcErr := svc.View(context.Background(), validToken, saved.ThingID)
	repoCall.Unset()

	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	esConfig, esErr := svc.View(context.Background(), validToken, saved.ThingID)
	repoCall.Unset()

	assert.Equal(t, svcConfig, esConfig, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfig, esConfig))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestUpdate(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	c := config

	ch := channel
	ch.ID = testsutil.GenerateUUID(t)
	c.Channels = append(c.Channels, ch)
	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: c.ThingID, Credentials: mgsdk.Credentials{Secret: c.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	modified := saved
	modified.Content = "new-config"
	modified.Name = "new name"

	nonExisting := config
	nonExisting.ThingID = "unknown"

	channels := []string{modified.Channels[0].ID, modified.Channels[1].ID}
	chs, err := json.Marshal(channels)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc   string
		config bootstrap.Config
		token  string
		err    error
		event  map[string]interface{}
	}{
		{
			desc:   "update config successfully",
			config: modified,
			token:  validToken,
			err:    nil,
			event: map[string]interface{}{
				"name":        modified.Name,
				"content":     modified.Content,
				"timestamp":   time.Now().UnixNano(),
				"operation":   configUpdate,
				"channels":    string(chs),
				"external_id": modified.ExternalID,
				"thing_id":    modified.ThingID,
				"owner":       validID,
				"state":       "0",
				"occurred_at": time.Now().UnixNano(),
			},
		},
		{
			desc:   "update non-existing config",
			config: nonExisting,
			token:  validToken,
			err:    svcerr.ErrNotFound,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
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
		repoCall.Unset()
	}
}

func TestUpdateConnections(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc        string
		id          string
		token       string
		connections []string
		err         error
		event       map[string]interface{}
	}{
		{
			desc:        "update connections successfully",
			id:          saved.ThingID,
			token:       validToken,
			connections: []string{config.Channels[0].ID},
			err:         nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"channels":  "2",
				"timestamp": time.Now().Unix(),
				"operation": thingUpdateConnections,
			},
		},
		{
			desc:        "update connections unsuccessfully",
			id:          saved.ThingID,
			token:       validToken,
			connections: []string{"256"},
			err:         svcerr.ErrNotFound,
			event:       nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
		repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
		repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateCert(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc       string
		id         string
		token      string
		clientCert string
		clientKey  string
		caCert     string
		err        error
		event      map[string]interface{}
	}{
		{
			desc:       "update cert successfully",
			id:         saved.ThingID,
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        nil,
			event: map[string]interface{}{
				"thing_key":   saved.ThingKey,
				"client_cert": "clientCert",
				"client_key":  "clientKey",
				"ca_cert":     "caCert",
				"operation":   certUpdate,
			},
		},
		{
			desc:       "invalid token",
			id:         saved.ThingID,
			token:      "invalid",
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        svcerr.ErrAuthentication,
			event:      nil,
		},
		{
			desc:       "invalid thing ID",
			id:         "invalidThingID",
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        svcerr.ErrNotFound,
			event:      nil,
		},
		{
			desc:       "empty client certificate",
			id:         saved.ThingID,
			token:      validToken,
			clientCert: "",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        nil,
			event:      nil,
		},
		{
			desc:       "empty client key",
			id:         saved.ThingID,
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "",
			caCert:     "caCert",
			err:        nil,
			event:      nil,
		},
		{
			desc:       "empty CA certificate",
			id:         saved.ThingID,
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "",
			err:        nil,
			event:      nil,
		},
		{
			desc:       "update cert with invalid token",
			id:         saved.ThingID,
			token:      "invalid",
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        svcerr.ErrAuthentication,
			event:      nil,
		},
		{
			desc:       "update cert with invalid thing ID",
			id:         "invalidThingID",
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "caCert",
			err:        svcerr.ErrNotFound,
			event:      nil,
		},
		{
			desc:       "successful update without CA certificate",
			id:         saved.ThingID,
			token:      validToken,
			clientCert: "clientCert",
			clientKey:  "clientKey",
			caCert:     "",
			err:        nil,
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
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)

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

		repoCall.Unset()
	}
}

func TestList(t *testing.T) {
	svc, auth, sdk := newService(t, redisURL)

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	_, err := svc.Add(context.Background(), validToken, config)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	offset := uint64(0)
	limit := uint64(10)
	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	svcConfigs, svcErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	repoCall.Unset()

	repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	esConfigs, esErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)
	repoCall.Unset()
	assert.Equal(t, svcConfigs, esConfigs)
	assert.Equal(t, svcErr, esErr)
}

func TestRemove(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	c := config

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(config.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "remove config successfully",
			id:    saved.ThingID,
			token: validToken,
			err:   nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"timestamp": time.Now().Unix(),
				"operation": configRemove,
			},
		},
		{
			desc:  "remove config with invalid credentials",
			id:    saved.ThingID,
			token: "",
			err:   svcerr.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID}, nil)
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
		repoCall.Unset()
	}
}

func TestBootstrap(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	c := config

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(mgsdk.Channel{}, nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc        string
		externalID  string
		externalKey string
		err         error
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
	}
}

func TestChangeState(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, auth, sdk := newService(t, redisURL)

	c := config

	repoCall := auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: validToken}).Return(&magistrala.IdentityRes{Id: validID}, nil)
	repoCall1 := sdk.On("Thing", mock.Anything, mock.Anything).Return(mgsdk.Thing{ID: config.ThingID, Credentials: mgsdk.Credentials{Secret: config.ThingKey}}, nil)
	repoCall2 := sdk.On("Channel", mock.Anything, mock.Anything).Return(toChannel(c.Channels[0]), nil)
	saved, err := svc.Add(context.Background(), validToken, c)
	assert.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	repoCall.Unset()
	repoCall1.Unset()
	repoCall2.Unset()

	err = redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		state bootstrap.State
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "change state to active",
			id:    saved.ThingID,
			token: validToken,
			state: bootstrap.Active,
			err:   nil,
			event: map[string]interface{}{
				"thing_id":  saved.ThingID,
				"state":     bootstrap.Active.String(),
				"timestamp": time.Now().Unix(),
				"operation": thingStateChange,
			},
		},
		{
			desc:  "change state invalid credentials",
			id:    saved.ThingID,
			token: "invalid",
			state: bootstrap.Inactive,
			err:   svcerr.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		repoCall = auth.On("Identify", mock.Anything, &magistrala.IdentityReq{Token: tc.token}).Return(&magistrala.IdentityRes{Id: validID, DomainId: testsutil.GenerateUUID(t)}, nil)
		repoCall1 = auth.On("Authorize", mock.Anything, mock.Anything).Return(&magistrala.AuthorizeRes{Authorized: true}, nil)
		repoCall2 = sdk.On("Connect", mock.Anything, mock.Anything).Return(nil)
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
		repoCall.Unset()
		repoCall1.Unset()
		repoCall2.Unset()
	}
}

func TestUpdateChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, _, _ := newService(t, redisURL)

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
	}
}

func TestRemoveChannelHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, _, _ := newService(t, redisURL)

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
	}
}

func TestRemoveConfigHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, _, _ := newService(t, redisURL)

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
	}
}

func TestDisconnectThingHandler(t *testing.T) {
	err := redisClient.FlushAll(context.Background()).Err()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	svc, _, _ := newService(t, redisURL)

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

		if expected["channels"] != nil || actual["channels"] != nil {
			ech := expected["channels"]
			ach := actual["channels"]

			che := []string{}
			err = json.Unmarshal([]byte(ech.(string)), &che)
			require.Nil(t, err, fmt.Sprintf("%s: expected to get a valid channels, got %s", description, err))

			cha := []string{}
			err = json.Unmarshal([]byte(ach.(string)), &cha)
			require.Nil(t, err, fmt.Sprintf("%s: expected to get a valid channels, got %s", description, err))

			if assert.ElementsMatchf(t, che, cha, "%s: got incorrect channels\n", description) {
				delete(expected, "channels")
				delete(actual, "channels")
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
