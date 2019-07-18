//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	wrongID    = ""
	wrongValue = "wrong-value"
	email      = "user@example.com"
	token      = "token"
)

var (
	thing   = things.Thing{Name: "test"}
	channel = things.Channel{Name: "test"}
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
	svc := newService(map[string]string{token: email})

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "add new thing",
			thing: things.Thing{Name: "a"},
			token: token,
			err:   nil,
		},
		{
			desc:  "add thing with wrong credentials",
			thing: things.Thing{Name: "d"},
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.AddThing(context.Background(), tc.token, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(context.Background(), token, thing)
	other := things.Thing{ID: wrongID, Key: "x"}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: saved,
			token: token,
			err:   nil,
		},
		{
			desc:  "update thing with wrong credentials",
			thing: saved,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "update non-existing thing",
			thing: other,
			token: token,
			err:   things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateThing(context.Background(), tc.token, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	key := "new-key"
	svc := newService(map[string]string{token: email})
	saved, err := svc.AddThing(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		token string
		id    string
		key   string
		err   error
	}{
		{
			desc:  "update key of an existing thing",
			token: token,
			id:    saved.ID,
			key:   key,
			err:   nil,
		},
		{
			desc:  "update key with invalid credentials",
			token: wrongValue,
			id:    saved.ID,
			key:   key,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "update key of non-existing thing",
			token: token,
			id:    wrongID,
			key:   wrongValue,
			err:   things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateKey(context.Background(), tc.token, tc.id, tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(context.Background(), token, thing)

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing thing": {
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		"view thing with wrong credentials": {
			id:    saved.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		"view non-existing thing": {
			id:    wrongID,
			token: token,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewThing(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		svc.AddThing(context.Background(), token, thing)
	}

	cases := map[string]struct {
		token  string
		offset uint64
		limit  uint64
		name   string
		size   uint64
		err    error
	}{
		"list all things": {
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			err:    nil,
		},
		"list half": {
			token:  token,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			err:    nil,
		},
		"list last thing": {
			token:  token,
			offset: n - 1,
			limit:  n,
			size:   1,
			err:    nil,
		},
		"list empty set": {
			token:  token,
			offset: n + 1,
			limit:  n,
			size:   0,
			err:    nil,
		},
		"list with zero limit": {
			token:  token,
			offset: 1,
			limit:  0,
			size:   0,
			err:    nil,
		},
		"list with wrong credentials": {
			token:  wrongValue,
			offset: 0,
			limit:  0,
			size:   0,
			err:    things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThings(context.Background(), tc.token, tc.offset, tc.limit, tc.name)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThingsByChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sch, err := svc.CreateChannel(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		sth, err := svc.AddThing(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		svc.Connect(context.Background(), token, sch.ID, sth.ID)
	}

	// Wait for things and channels to connect
	time.Sleep(time.Second)

	cases := map[string]struct {
		token   string
		channel string
		offset  uint64
		limit   uint64
		size    uint64
		err     error
	}{
		"list all things by existing channel": {
			token:   token,
			channel: sch.ID,
			offset:  0,
			limit:   n,
			size:    n,
			err:     nil,
		},
		"list half of things by existing channel": {
			token:   token,
			channel: sch.ID,
			offset:  n / 2,
			limit:   n,
			size:    n / 2,
			err:     nil,
		},
		"list last thing by existing channel": {
			token:   token,
			channel: sch.ID,
			offset:  n - 1,
			limit:   n,
			size:    1,
			err:     nil,
		},
		"list empty set of things by existing channel": {
			token:   token,
			channel: sch.ID,
			offset:  n + 1,
			limit:   n,
			size:    0,
			err:     nil,
		},
		"list things by existing channel with zero limit": {
			token:   token,
			channel: sch.ID,
			offset:  1,
			limit:   0,
			size:    0,
			err:     nil,
		},
		"list things by existing channel with wrong credentials": {
			token:   wrongValue,
			channel: sch.ID,
			offset:  0,
			limit:   0,
			size:    0,
			err:     things.ErrUnauthorizedAccess,
		},
		"list things by non-existent channel with wrong credentials": {
			token:   token,
			channel: "non-existent",
			offset:  0,
			limit:   10,
			size:    0,
			err:     nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThingsByChannel(context.Background(), tc.token, tc.channel, tc.offset, tc.limit)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(context.Background(), token, thing)

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove thing with wrong credentials",
			id:    saved.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "remove existing thing",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed thing",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing thing",
			id:    wrongID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveThing(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := []struct {
		desc    string
		channel things.Channel
		token   string
		err     error
	}{
		{
			desc:    "create channel",
			channel: channel,
			token:   token,
			err:     nil,
		},
		{
			desc:    "create channel with wrong credentials",
			channel: channel,
			token:   wrongValue,
			err:     things.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateChannel(context.Background(), tc.token, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(context.Background(), token, channel)
	other := things.Channel{ID: wrongID}

	cases := []struct {
		desc    string
		channel things.Channel
		token   string
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: saved,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update channel with wrong credentials",
			channel: saved,
			token:   wrongValue,
			err:     things.ErrUnauthorizedAccess,
		},
		{
			desc:    "update non-existing channel",
			channel: other,
			token:   token,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateChannel(context.Background(), tc.token, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(context.Background(), token, channel)

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing channel": {
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		"view channel with wrong credentials": {
			id:    saved.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		"view non-existing channel": {
			id:    wrongID,
			token: token,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		svc.CreateChannel(context.Background(), token, channel)
	}
	cases := map[string]struct {
		token  string
		offset uint64
		limit  uint64
		size   uint64
		name   string
		err    error
	}{
		"list all channels": {
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			err:    nil,
		},
		"list half": {
			token:  token,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			err:    nil,
		},
		"list last channel": {
			token:  token,
			offset: n - 1,
			limit:  n,
			size:   1,
			err:    nil,
		},
		"list empty set": {
			token:  token,
			offset: n + 1,
			limit:  n,
			size:   0,
			err:    nil,
		},
		"list with zero limit": {
			token:  token,
			offset: 1,
			limit:  0,
			size:   0,
			err:    nil,
		},
		"list with wrong credentials": {
			token:  wrongValue,
			offset: 0,
			limit:  0,
			size:   0,
			err:    things.ErrUnauthorizedAccess,
		},
		"list with existing name": {
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			name:   "chanel_name",
			err:    nil,
		},
		"list with non-existent name": {
			token:  token,
			offset: 0,
			limit:  n,
			size:   n,
			name:   "wrong",
			err:    nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannels(context.Background(), tc.token, tc.offset, tc.limit, tc.name)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannelsByThing(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, err := svc.AddThing(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		sch, err := svc.CreateChannel(context.Background(), token, channel)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		svc.Connect(context.Background(), token, sch.ID, sth.ID)
	}

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := map[string]struct {
		token  string
		thing  string
		offset uint64
		limit  uint64
		size   uint64
		err    error
	}{
		"list all channels by existing thing": {
			token:  token,
			thing:  sth.ID,
			offset: 0,
			limit:  n,
			size:   n,
			err:    nil,
		},
		"list half of channels by existing thing": {
			token:  token,
			thing:  sth.ID,
			offset: n / 2,
			limit:  n,
			size:   n / 2,
			err:    nil,
		},
		"list last channel by existing thing": {
			token:  token,
			thing:  sth.ID,
			offset: n - 1,
			limit:  n,
			size:   1,
			err:    nil,
		},
		"list empty set of channels by existing thing": {
			token:  token,
			thing:  sth.ID,
			offset: n + 1,
			limit:  n,
			size:   0,
			err:    nil,
		},
		"list channels by existing thing with zero limit": {
			token:  token,
			thing:  sth.ID,
			offset: 1,
			limit:  0,
			size:   0,
			err:    nil,
		},
		"list channels by existing thing with wrong credentials": {
			token:  wrongValue,
			thing:  sth.ID,
			offset: 0,
			limit:  0,
			size:   0,
			err:    things.ErrUnauthorizedAccess,
		},
		"list channels by non-existent thing": {
			token:  token,
			thing:  "non-existent",
			offset: 0,
			limit:  10,
			size:   0,
			err:    nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannelsByThing(context.Background(), tc.token, tc.thing, tc.offset, tc.limit)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(context.Background(), token, channel)

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove channel with wrong credentials",
			id:    saved.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "remove existing channel",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed channel",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing channel",
			id:    saved.ID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveChannel(context.Background(), tc.token, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)

	cases := []struct {
		desc    string
		token   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "connect thing",
			token:   token,
			chanID:  sch.ID,
			thingID: sth.ID,
			err:     nil,
		},
		{
			desc:    "connect thing with wrong credentials",
			token:   wrongValue,
			chanID:  sch.ID,
			thingID: sth.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		{
			desc:    "connect thing to non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: sth.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "connect non-existing thing to channel",
			token:   token,
			chanID:  sch.ID,
			thingID: wrongID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Connect(context.Background(), tc.token, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)
	svc.Connect(context.Background(), token, sch.ID, sth.ID)

	cases := []struct {
		desc    string
		token   string
		chanID  string
		thingID string
		err     error
	}{
		{
			desc:    "disconnect connected thing",
			token:   token,
			chanID:  sch.ID,
			thingID: sth.ID,
			err:     nil,
		},
		{
			desc:    "disconnect disconnected thing",
			token:   token,
			chanID:  sch.ID,
			thingID: sth.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect with wrong credentials",
			token:   wrongValue,
			chanID:  sch.ID,
			thingID: sth.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		{
			desc:    "disconnect from non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: sth.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing thing",
			token:   token,
			chanID:  sch.ID,
			thingID: wrongID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Disconnect(context.Background(), tc.token, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccess(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)
	svc.Connect(context.Background(), token, sch.ID, sth.ID)

	cases := map[string]struct {
		token   string
		channel string
		err     error
	}{
		"allowed access": {
			token:   sth.Key,
			channel: sch.ID,
			err:     nil,
		},
		"not-connected cannot access": {
			token:   wrongValue,
			channel: sch.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		"access to non-existing channel": {
			token:   sth.Key,
			channel: wrongID,
			err:     things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccess(context.Background(), tc.channel, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCanAccessByID(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(context.Background(), token, thing)
	sch, _ := svc.CreateChannel(context.Background(), token, channel)
	svc.Connect(context.Background(), token, sch.ID, sth.ID)

	cases := map[string]struct {
		thingID string
		channel string
		err     error
	}{
		"allowed access": {
			thingID: sth.ID,
			channel: sch.ID,
			err:     nil,
		},
		"not-connected cannot access": {
			thingID: wrongValue,
			channel: sch.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		"access to non-existing channel": {
			thingID: sth.ID,
			channel: wrongID,
			err:     things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		err := svc.CanAccessByID(context.Background(), tc.channel, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(context.Background(), token, thing)

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify existing thing": {
			token: sth.Key,
			id:    sth.ID,
			err:   nil,
		},
		"identify non-existing thing": {
			token: wrongValue,
			id:    wrongID,
			err:   things.ErrUnauthorizedAccess,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.token)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
