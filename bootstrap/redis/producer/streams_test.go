//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package producer_test

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/mainflux/mainflux"

	"github.com/mainflux/mainflux/bootstrap"
	"github.com/mainflux/mainflux/bootstrap/mocks"
	"github.com/mainflux/mainflux/bootstrap/redis/producer"
	mfsdk "github.com/mainflux/mainflux/sdk/go"
	"github.com/mainflux/mainflux/things"
	httpapi "github.com/mainflux/mainflux/things/api/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	streamID    = "mainflux.bootstrap"
	email       = "user@example.com"
	validToken  = "validToken"
	unknownID   = "1"
	unknownKey  = "2"
	channelsNum = 3

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

func newService(users mainflux.UsersServiceClient, url string) bootstrap.Service {
	configs := mocks.NewConfigsRepository(map[string]string{unknownID: unknownKey})
	config := mfsdk.Config{
		BaseURL: url,
	}

	sdk := mfsdk.NewSDK(config)
	return bootstrap.New(users, configs, sdk)
}

func newThingsService(users mainflux.UsersServiceClient) things.Service {
	channels := make(map[string]things.Channel, channelsNum)
	for i := 0; i < channelsNum; i++ {
		id := strconv.Itoa(i + 1)
		channels[id] = things.Channel{
			ID:       id,
			Owner:    email,
			Metadata: map[string]interface{}{"meta": "data"},
		}
	}

	return mocks.NewThingsService(map[string]things.Thing{}, channels, users)
}

func newThingsServer(svc things.Service) *httptest.Server {
	mux := httpapi.MakeHandler(svc)
	return httptest.NewServer(mux)
}
func TestAdd(t *testing.T) {
	redisClient.FlushAll().Err()
	users := mocks.NewUsersService(map[string]string{validToken: email})

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
		key    string
		err    error
		event  map[string]interface{}
	}{
		{
			desc:   "create config successfully",
			config: config,
			key:    validToken,
			err:    nil,
			event: map[string]interface{}{
				"thing_id":    "1",
				"owner":       email,
				"name":        config.Name,
				"channels":    strings.Join(channels, ", "),
				"external_id": config.ExternalID,
				"content":     config.Content,
				"timestamp":   strconv.FormatInt(time.Now().Unix(), 10),
				"operation":   configCreate,
			},
		},
		{
			desc:   "create invalid config",
			config: invalidConfig,
			key:    validToken,
			err:    bootstrap.ErrMalformedEntity,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.Add(tc.key, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestView(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	saved, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	svcConfig, svcErr := svc.View(validToken, saved.MFThing)

	svc = producer.NewEventStoreMiddleware(svc, redisClient)
	esConfig, esErr := svc.View(validToken, saved.MFThing)

	assert.Equal(t, svcConfig, esConfig, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfig, esConfig))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestUpdate(t *testing.T) {
	redisClient.FlushAll().Err()

	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	ch := channel
	ch.ID = "2"
	c.MFChannels = append(c.MFChannels, ch)
	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll().Err()

	modified := saved
	modified.Content = "new-config"
	modified.Name = "new name"

	nonExisting := config
	nonExisting.MFThing = "unknown"

	cases := []struct {
		desc   string
		config bootstrap.Config
		key    string
		err    error
		event  map[string]interface{}
	}{
		{
			desc:   "update config successfully",
			config: modified,
			key:    validToken,
			err:    nil,
			event: map[string]interface{}{
				"thing_id":  modified.MFThing,
				"name":      modified.Name,
				"content":   modified.Content,
				"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"operation": configUpdate,
			},
		},
		{
			desc:   "update non-existing config",
			config: nonExisting,
			key:    validToken,
			err:    bootstrap.ErrNotFound,
			event:  nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Update(tc.key, tc.config)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestUpdateConnections(t *testing.T) {
	redisClient.FlushAll().Err()

	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	saved, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll().Err()

	cases := []struct {
		desc        string
		id          string
		key         string
		connections []string
		err         error
		event       map[string]interface{}
	}{
		{
			desc:        "update connections successfully",
			id:          saved.MFThing,
			key:         validToken,
			connections: []string{"2"},
			err:         nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"channels":  "2",
				"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"operation": thingUpdateConnections,
			},
		},
		{
			desc:        "update connections unsuccessfully",
			id:          saved.MFThing,
			key:         validToken,
			connections: []string{"256"},
			err:         bootstrap.ErrMalformedEntity,
			event:       nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateConnections(tc.key, tc.id, tc.connections)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}
func TestList(t *testing.T) {
	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)

	_, err := svc.Add(validToken, config)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))

	offset := uint64(0)
	limit := uint64(10)
	svcConfigs, svcErr := svc.List(validToken, bootstrap.Filter{}, offset, limit)

	svc = producer.NewEventStoreMiddleware(svc, redisClient)
	esConfigs, esErr := svc.List(validToken, bootstrap.Filter{}, offset, limit)

	assert.Equal(t, svcConfigs, esConfigs, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcConfigs, esConfigs))
	assert.Equal(t, svcErr, esErr, fmt.Sprintf("event sourcing changed service behavior: expected %v got %v", svcErr, esErr))
}

func TestRemove(t *testing.T) {
	redisClient.FlushAll().Err()

	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll().Err()

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "remove config successfully",
			id:   saved.MFThing,
			key:  validToken,
			err:  nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"operation": configRemove,
			},
		},
		{
			desc:  "remove config with invalid credentials",
			id:    saved.MFThing,
			key:   "",
			err:   bootstrap.ErrUnauthorizedAccess,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Remove(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestBootstrap(t *testing.T) {
	redisClient.FlushAll().Err()

	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll().Err()

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
				"timestamp":   strconv.FormatInt(time.Now().Unix(), 10),
				"operation":   thingBootstrap,
			},
		},
		{
			desc:        "bootstrap with an error",
			externalID:  saved.ExternalID,
			externalKey: "external",
			err:         bootstrap.ErrNotFound,
			event: map[string]interface{}{
				"external_id": saved.ExternalID,
				"success":     "0",
				"timestamp":   strconv.FormatInt(time.Now().Unix(), 10),
				"operation":   thingBootstrap,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.Bootstrap(tc.externalKey, tc.externalID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}

func TestChangeState(t *testing.T) {
	redisClient.FlushAll().Err()

	users := mocks.NewUsersService(map[string]string{validToken: email})
	server := newThingsServer(newThingsService(users))
	svc := newService(users, server.URL)
	svc = producer.NewEventStoreMiddleware(svc, redisClient)

	c := config

	saved, err := svc.Add(validToken, c)
	require.Nil(t, err, fmt.Sprintf("Saving config expected to succeed: %s.\n", err))
	redisClient.FlushAll().Err()

	cases := []struct {
		desc  string
		id    string
		key   string
		state bootstrap.State
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "change state to active",
			id:    saved.MFThing,
			key:   validToken,
			state: bootstrap.Active,
			err:   nil,
			event: map[string]interface{}{
				"thing_id":  saved.MFThing,
				"state":     bootstrap.Active.String(),
				"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
				"operation": thingStateChange,
			},
		},
		{
			desc:  "change state invalid credentials",
			id:    saved.MFThing,
			key:   "",
			state: bootstrap.Inactive,
			err:   bootstrap.ErrUnauthorizedAccess,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.ChangeState(tc.key, tc.id, tc.state)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&redis.XReadArgs{
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

		assert.Equal(t, tc.event, event, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.event, event))
	}
}
