//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis_test

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	r "github.com/go-redis/redis"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/mainflux/mainflux/things/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	streamID        = "mainflux.things"
	email           = "user@example.com"
	token           = "token"
	thingPrefix     = "thing."
	thingCreate     = thingPrefix + "create"
	thingUpdate     = thingPrefix + "update"
	thingRemove     = thingPrefix + "remove"
	thingConnect    = thingPrefix + "connect"
	thingDisconnect = thingPrefix + "disconnect"

	channelPrefix = "channel."
	channelCreate = channelPrefix + "create"
	channelUpdate = channelPrefix + "update"
	channelRemove = channelPrefix + "remove"
)

func newService(tokens map[string]string) things.Service {
	users := mocks.NewUsersService(tokens)
	thingsRepo := mocks.NewThingRepository()
	channelsRepo := mocks.NewChannelRepository(thingsRepo)
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idp := mocks.NewIdentityProvider()

	return things.New(users, thingsRepo, channelsRepo, chanCache, thingCache, idp)
}

func TestAddThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		thing things.Thing
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "create thing successfully",
			thing: things.Thing{Type: "app", Name: "a", Metadata: "metadata"},
			key:   token,
			err:   nil,
			event: map[string]interface{}{
				"id":        "1",
				"name":      "a",
				"owner":     email,
				"type":      "app",
				"metadata":  "metadata",
				"operation": thingCreate,
			},
		},
		{
			desc:  "create invalid thing",
			thing: things.Thing{Type: "a", Name: "a"},
			key:   token,
			err:   things.ErrMalformedEntity,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.AddThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestUpdateThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(token, things.Thing{Type: "app", Name: "a", Metadata: "metadata"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		thing things.Thing
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "update existing thing successfully",
			thing: things.Thing{ID: sth.ID, Type: "app", Name: "a", Metadata: "metadata1"},
			key:   token,
			err:   nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"name":      "a",
				"type":      "app",
				"metadata":  "metadata1",
				"operation": thingUpdate,
			},
		},
		{
			desc: "update invalid thing",
			thing: things.Thing{
				ID:   strconv.FormatUint(math.MaxUint64, 10),
				Type: "a",
				Name: "a",
			},
			key:   token,
			err:   things.ErrMalformedEntity,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestRemoveThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(token, things.Thing{Type: "app", Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "delete existing thing successfully",
			id:   sth.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"operation": thingRemove,
			},
		},
		{
			desc:  "delete thing with invalid credentials",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   things.ErrUnauthorizedAccess,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveThing(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestCreateChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		channel things.Channel
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "create channel successfully",
			channel: things.Channel{Name: "a", Metadata: "metadata"},
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"id":        "1",
				"name":      "a",
				"metadata":  "metadata",
				"owner":     email,
				"operation": channelCreate,
			},
		},
		{
			desc:    "create channel with invalid credentials",
			channel: things.Channel{Name: "a", Metadata: "metadata"},
			key:     "",
			err:     things.ErrUnauthorizedAccess,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestUpdateChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	sch, err := svc.CreateChannel(token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		channel things.Channel
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "update channel successfully",
			channel: things.Channel{ID: sch.ID, Name: "b", Metadata: "metadata"},
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"id":        sch.ID,
				"name":      "b",
				"metadata":  "metadata",
				"operation": channelUpdate,
			},
		},
		{
			desc: "create non-existent channel",
			channel: things.Channel{
				ID:   strconv.FormatUint(math.MaxUint64, 10),
				Name: "c",
			},
			key:   token,
			err:   things.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestRemoveChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	sch, err := svc.CreateChannel(token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		key   string
		err   error
		event map[string]interface{}
	}{
		{
			desc: "update channel successfully",
			id:   sch.ID,
			key:  token,
			err:  nil,
			event: map[string]interface{}{
				"id":        sch.ID,
				"operation": channelRemove,
			},
		},
		{
			desc:  "create non-existent channel",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			key:   "",
			err:   things.ErrUnauthorizedAccess,
			event: nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestConnectEvent(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing and channel that will be connected.
	sth, err := svc.AddThing(token, things.Thing{Type: "device", Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "connect existing thing to existing channel",
			thingID: sth.ID,
			chanID:  sch.ID,
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"chan_id":   sch.ID,
				"thing_id":  sth.ID,
				"operation": thingConnect,
			},
		},
		{
			desc:    "connect non-existent thing to channel",
			thingID: strconv.FormatUint(math.MaxUint64, 10),
			chanID:  sch.ID,
			key:     token,
			err:     things.ErrNotFound,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Connect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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

func TestDisconnectEvent(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing and channel that will be connected.
	sth, err := svc.AddThing(token, things.Thing{Type: "device", Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	err = svc.Connect(token, sch.ID, sth.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc    string
		thingID string
		chanID  string
		key     string
		err     error
		event   map[string]interface{}
	}{
		{
			desc:    "disconnect thing from channel",
			thingID: sth.ID,
			chanID:  sch.ID,
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"chan_id":   sch.ID,
				"thing_id":  sth.ID,
				"operation": thingDisconnect,
			},
		},
		{
			desc:    "disconnect non-existent thing from channel",
			thingID: strconv.FormatUint(math.MaxUint64, 10),
			chanID:  sch.ID,
			key:     token,
			err:     things.ErrNotFound,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.Disconnect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
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
