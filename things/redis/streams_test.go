//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package redis_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

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
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
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
			desc: "create thing successfully",
			thing: things.Thing{
				Name:     "a",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        "1",
				"name":      "a",
				"owner":     email,
				"metadata":  "{\"test\":\"test\"}",
				"operation": thingCreate,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.AddThing(context.Background(), tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestUpdateThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	th := things.Thing{Name: "a", Metadata: map[string]interface{}{"test": "test"}}
	sth, err := svc.AddThing(context.Background(), token, th)
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
			desc: "update existing thing successfully",
			thing: things.Thing{
				ID:       sth.ID,
				Name:     "a",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        sth.ID,
				"name":      "a",
				"metadata":  "{\"test\":\"test\"}",
				"operation": thingUpdate,
			},
		},
	}

	lastID := "0"
	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestViewThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esth, eserr := essvc.ViewThing(context.Background(), token, sth.ID)
	th, err := svc.ViewThing(context.Background(), token, sth.ID)
	assert.Equal(t, th, esth, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", th, esth))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestListThings(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	_, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThings(context.Background(), token, 0, 10, "")
	ths, err := svc.ListThings(context.Background(), token, 0, 10, "")
	assert.Equal(t, ths, esths, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", ths, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestListThingsByChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	err = svc.Connect(context.Background(), token, sch.ID, sth.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esths, eserr := essvc.ListThingsByChannel(context.Background(), token, sch.ID, 0, 10)
	ths, err := svc.ListThingsByChannel(context.Background(), token, sch.ID, 0, 10)
	assert.Equal(t, ths, esths, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", ths, esths))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestRemoveThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
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
		err := svc.RemoveThing(context.Background(), tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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
			channel: things.Channel{Name: "a", Metadata: map[string]interface{}{"test": "test"}},
			key:     token,
			err:     nil,
			event: map[string]interface{}{
				"id":        "1",
				"name":      "a",
				"metadata":  "{\"test\":\"test\"}",
				"owner":     email,
				"operation": channelCreate,
			},
		},
		{
			desc:    "create channel with invalid credentials",
			channel: things.Channel{Name: "a", Metadata: map[string]interface{}{"test": "test"}},
			key:     "",
			err:     things.ErrUnauthorizedAccess,
			event:   nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		_, err := svc.CreateChannel(context.Background(), tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestUpdateChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
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
			desc: "update channel successfully",
			channel: things.Channel{
				ID:       sch.ID,
				Name:     "b",
				Metadata: map[string]interface{}{"test": "test"},
			},
			key: token,
			err: nil,
			event: map[string]interface{}{
				"id":        sch.ID,
				"name":      "b",
				"metadata":  "{\"test\":\"test\"}",
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
		err := svc.UpdateChannel(context.Background(), tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestViewChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	esch, eserr := essvc.ViewChannel(context.Background(), token, sch.ID)
	ch, err := svc.ViewChannel(context.Background(), token, sch.ID)
	assert.Equal(t, ch, esch, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", ch, esch))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestListChannels(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	_, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	eschs, eserr := essvc.ListChannels(context.Background(), token, 0, 10, "")
	chs, err := svc.ListChannels(context.Background(), token, 0, 10, "")
	assert.Equal(t, chs, eschs, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", chs, eschs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestListChannelsByThing(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing without sending event.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	err = svc.Connect(context.Background(), token, sch.ID, sth.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))

	essvc := redis.NewEventStoreMiddleware(svc, redisClient)
	eschs, eserr := essvc.ListChannelsByThing(context.Background(), token, sth.ID, 0, 10)
	chs, err := svc.ListChannelsByThing(context.Background(), token, sth.ID, 0, 10)
	assert.Equal(t, chs, eschs, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", chs, eschs))
	assert.Equal(t, err, eserr, fmt.Sprintf("event sourcing changed service behaviour: expected %v got %v", err, eserr))
}

func TestRemoveChannel(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create channel without sending event.
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
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
		err := svc.RemoveChannel(context.Background(), tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestConnectEvent(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing and channel that will be connected.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
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
		err := svc.Connect(context.Background(), tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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

func TestDisconnectEvent(t *testing.T) {
	redisClient.FlushAll().Err()

	svc := newService(map[string]string{token: email})
	// Create thing and channel that will be connected.
	sth, err := svc.AddThing(context.Background(), token, things.Thing{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	sch, err := svc.CreateChannel(context.Background(), token, things.Channel{Name: "a"})
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	err = svc.Connect(context.Background(), token, sch.ID, sth.ID)
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
		err := svc.Disconnect(context.Background(), tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))

		streams := redisClient.XRead(&r.XReadArgs{
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
