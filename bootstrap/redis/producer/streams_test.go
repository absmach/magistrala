// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package producer_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go/mocktracer"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/bootstrap/redis/producer"
	mfsdk "github.com/mainflux/mainflux/pkg/sdk/go"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/things/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	streamID      = "mainflux.bootstrap"
	email         = "user@example.com"
	validToken    = "validToken"
	channelsNum   = 3
	defaultTimout = 5

	configPrefix = "config."
	configCreate = configPrefix + "create"
	configUpdate = configPrefix + "update"
	configRemove = configPrefix + "remove"

	thingPrefix            = "thing."
	thingStateChange       = thingPrefix + "state_change"
	thingBootstrap         = thingPrefix + "bootstrap"
	thingUpdateConnections = thingPrefix + "update_connections"
)

var (
	encKey = []byte("1234567891011121")

	channel = bootstrap.Channel{
		ID:       "1",
		Name:     "name",
		Metadata: map[string]interface{}{"name": "value"},
	}

	config = bootstrap.Config{
		ExternalID:  "external_id",
		ExternalKey: "external_key",
		MFChannels:  []bootstrap.Channel{channel},
		Content:     "config",
	}
)

func newService(auth mainflux.AuthServiceClient, url string) bootstrap.Service {
	configs := mocks.NewConfigsRepository()
	config := mfsdk.Config{
		ThingsURL: url,
	}

	sdk := mfsdk.NewSDK(config)
	return bootstrap.New(auth, configs, sdk, encKey)
}

func newThingsService(auth mainflux.AuthServiceClient) things.Service {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: map[string]interface{}{"meta": "data"},
		}
	}

	return mocks.NewThingsService(map[string]things.Thing{}, channels, auth)
}

func newThingsServer(svc things.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}
func TestAdd(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()
	users := mocks.NewAuthClient(map[string]string{validToken: email})

	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	var channels []string
	for _, ch := range config.MFChannels {
		channels = append(channels, ch.ID)
	}

	invalidConfig := config
	invalidConfig.MFChannels = []bootstrap.Channel{bootstrap.Channel{ID: "empty"}}

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
			err:    errors.ErrMalformedEntity,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.Add(context.Background(), tc.token, tc.config)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}

func TestView(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	svcConfig, svcErr := svc.View(context.Background(), validToken, saved.MFThing)

	svc = producer.NewEventStoreMiddleware(svc, redisClient)
	esConfig, esErr := svc.View(context.Background(), validToken, saved.MFThing)

	assert.Equal(t, svcConfig, esConfig, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfig, esConfig))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestUpdate(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()

	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll(context.Background()).Err()

	modified := saved
	modified.Content = "new-config"
	modified.Name = "new name"

	nonExisting := config
	nonExisting.MFThing = "unknown"

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
				"thing_id":  modified.MFThing,
				"name":      modified.Name,
				"content":   modified.Content,
				"timestamp": time.Now().Unix(),
				"operation": configUpdate,
			},
		},
		{
			desc:   "update non-existing config",
			config: nonExisting,
			token:  validToken,
			err:    errors.ErrNotFound,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Update(context.Background(), tc.token, tc.config)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}

func TestUpdateConnections(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()

	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	saved, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll(context.Background()).Err()

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
			id:          saved.MFThing,
			token:       validToken,
			connections: []string{"2"},
			err:         nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"channels":  "2",
				"timestamp": time.Now().Unix(),
				"operation": thingUpdateConnections,
			},
		},
		{
			desc:        "update connections unsuccessfully",
			id:          saved.MFThing,
			token:       validToken,
			connections: []string{"256"},
			err:         errors.ErrMalformedEntity,
			event:       nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateConnections(context.Background(), tc.token, tc.id, tc.connections)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}
func TestList(t *testing.T) {
	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(context.Background(), validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	offset := uint64(0)
	limit := uint64(10)
	svcConfigs, svcErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)

	svc = producer.NewEventStoreMiddleware(svc, redisClient)
	esConfigs, esErr := svc.List(context.Background(), validToken, bootstrap.Filter{}, offset, limit)

	assert.Equal(t, svcConfigs, esConfigs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfigs, esConfigs))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestRemove(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()

	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll(context.Background()).Err()

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "remove config successfully",
			id:    saved.MFThing,
			token: validToken,
			err:   nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"timestamp": time.Now().Unix(),
				"operation": configRemove,
			},
		},
		{
			desc:  "remove config with invalid credentials",
			id:    saved.MFThing,
			token: "",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Remove(context.Background(), tc.token, tc.id)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}

func TestBootstrap(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()

	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll(context.Background()).Err()

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
			externalID:  saved.ExternalID,
			externalKey: "external",
			err:         bootstrap.ErrExternalKey,
			event: map[string]interface{}{
				"external_id": saved.ExternalID,
				"success":     "0",
				"timestamp":   time.Now().Unix(),
				"operation":   thingBootstrap,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.Bootstrap(context.Background(), tc.externalKey, tc.externalID, false)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}

func TestChangeState(t *testing.T) {
	redisClient.FlushAll(context.Background()).Err()

	users := mocks.NewAuthClient(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(context.Background(), validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll(context.Background()).Err()

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
			id:    saved.MFThing,
			token: validToken,
			state: bootstrap.Active,
			err:   nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"state":     bootstrap.Active.String(),
				"timestamp": time.Now().Unix(),
				"operation": thingStateChange,
			},
		},
		{
			desc:  "change state invalid credentials",
			id:    saved.MFThing,
			token: "",
			state: bootstrap.Inactive,
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.ChangeState(context.Background(), tc.token, tc.id, tc.state)
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
			lastID = msg.ID
		}

		test(t, tc.event, event, tc.desc)
	}
}

func test(t *testing.T, expected, actual map[string]interface{}, description string) {
	if expected != nil && actual != nil {
		ts1 := expected["timestamp"].(int64)
		ts2, err := strconv.ParseInt(actual["timestamp"].(string), 10, 64)
		require.Nil(t, err, fmt.Sprintf("%s: expected to get a valid timestamp, got %s", description, err))
		val := ts1 == ts2 || ts2 <= ts1+defaultTimout
		assert.True(t, val, fmt.Sprintf("%s: timestamp is not in valid range", description))
		delete(expected, "timestamp")
		delete(actual, "timestamp")
		assert.Equal(t, expected, actual, fmt.Sprintf("%s: expected %v got %v\n", description, expected, actual))
	}
}
