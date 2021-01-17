// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/uuid"
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
	auth := mocks.NewAuthService(tokens)
	conns := make(chan mocks.Connection)
	thingsRepo := mocks.NewThingRepository(conns)
	channelsRepo := mocks.NewChannelRepository(thingsRepo, conns)
	chanCache := mocks.NewChannelCache()
	thingCache := mocks.NewThingCache()
	idProvider := uuid.NewMock()

	return things.New(auth, thingsRepo, channelsRepo, nil, chanCache, thingCache, idProvider)
}

func TestCreateThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := []struct {
		desc   string
		things []things.Thing
		token  string
		err    error
	}{
		{
			desc:   "create new things",
			things: []things.Thing{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			token:  token,
			err:    nil,
		},
		{
			desc:   "create thing with wrong credentials",
			things: []things.Thing{{Name: "e"}},
			token:  wrongValue,
			err:    things.ErrUnauthorizedAccess,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateThings(context.Background(), tc.token, tc.things...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	other := things.Thing{ID: wrongID, Key: "x"}

	cases := []struct {
		desc  string
		thing things.Thing
		token string
		err   error
	}{
		{
			desc:  "update existing thing",
			thing: th,
			token: token,
			err:   nil,
		},
		{
			desc:  "update thing with wrong credentials",
			thing: th,
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateKey(t *testing.T) {
	key := "new-key"
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

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
			id:    th.ID,
			key:   key,
			err:   nil,
		},
		{
			desc:  "update key with invalid credentials",
			token: wrongValue,
			id:    th.ID,
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing thing": {
			id:    th.ID,
			token: token,
			err:   nil,
		},
		"view thing with wrong credentials": {
			id:    th.ID,
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

	m := make(map[string]interface{})
	m["serial"] = "123456"
	thing.Metadata = m

	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		_, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		token        string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all things": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		"list last thing": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list with zero limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 1,
				Limit:  0,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  things.ErrUnauthorizedAccess,
		},
		"list with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: m,
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThings(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThingsByChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	n := uint64(10)
	thsDisconNum := uint64(1)

	for i := uint64(0); i < n; i++ {
		ths, err := svc.CreateThings(context.Background(), token, thing)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		th := ths[0]

		// Don't connect last Channel
		if i == n-thsDisconNum {
			break
		}

		err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	// Wait for things and channels to connect
	time.Sleep(time.Second)

	cases := map[string]struct {
		token     string
		channel   string
		offset    uint64
		limit     uint64
		connected bool
		size      uint64
		err       error
	}{
		"list all things by existing channel": {
			token:     token,
			channel:   ch.ID,
			offset:    0,
			limit:     n,
			connected: true,
			size:      n - thsDisconNum,
			err:       nil,
		},
		"list half of things by existing channel": {
			token:     token,
			channel:   ch.ID,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      (n / 2) - thsDisconNum,
			err:       nil,
		},
		"list last thing by existing channel": {
			token:     token,
			channel:   ch.ID,
			offset:    n - 1 - thsDisconNum,
			limit:     n,
			connected: true,
			size:      1,
			err:       nil,
		},
		"list empty set of things by existing channel": {
			token:     token,
			channel:   ch.ID,
			offset:    n + 1,
			limit:     n,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list things by existing channel with zero limit": {
			token:     token,
			channel:   ch.ID,
			offset:    1,
			limit:     0,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list things by existing channel with wrong credentials": {
			token:     wrongValue,
			channel:   ch.ID,
			offset:    0,
			limit:     0,
			connected: true,
			size:      0,
			err:       things.ErrUnauthorizedAccess,
		},
		"list things by non-existent channel with wrong credentials": {
			token:     token,
			channel:   "non-existent",
			offset:    0,
			limit:     10,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list all non connected things by existing channel": {
			token:     token,
			channel:   ch.ID,
			offset:    0,
			limit:     n,
			connected: false,
			size:      thsDisconNum,
			err:       nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListThingsByChannel(context.Background(), tc.token, tc.channel, tc.offset, tc.limit, tc.connected)
		size := uint64(len(page.Things))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sth := ths[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove thing with wrong credentials",
			id:    sth.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "remove existing thing",
			id:    sth.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed thing",
			id:    sth.ID,
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := []struct {
		desc     string
		channels []things.Channel
		token    string
		err      error
	}{
		{
			desc:     "create new channels",
			channels: []things.Channel{{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}},
			token:    token,
			err:      nil,
		},
		{
			desc:     "create channel with wrong credentials",
			channels: []things.Channel{{Name: "e"}},
			token:    wrongValue,
			err:      things.ErrUnauthorizedAccess,
		},
	}

	for _, cc := range cases {
		_, err := svc.CreateChannels(context.Background(), cc.token, cc.channels...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	other := things.Channel{ID: wrongID}

	cases := []struct {
		desc    string
		channel things.Channel
		token   string
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: ch,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update channel with wrong credentials",
			channel: ch,
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
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := map[string]struct {
		id       string
		token    string
		err      error
		metadata things.Metadata
	}{
		"view existing channel": {
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		"view channel with wrong credentials": {
			id:    ch.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		"view non-existing channel": {
			id:    wrongID,
			token: token,
			err:   things.ErrNotFound,
		},
		"view channel with metadata": {
			id:    wrongID,
			token: token,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})
	meta := things.Metadata{}
	meta["name"] = "test-channel"
	channel.Metadata = meta
	n := uint64(10)
	for i := uint64(0); i < n; i++ {
		svc.CreateChannels(context.Background(), token, channel)
	}
	cases := map[string]struct {
		token        string
		pageMetadata things.PageMetadata
		size         uint64
		err          error
	}{
		"list all channels": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		"list half": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		"list last channel": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		"list empty set": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: n + 1,
				Limit:  n,
			},
			size: 0,
			err:  nil,
		},
		"list with zero limit": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 1,
				Limit:  0,
			},
			size: 0,
			err:  nil,
		},
		"list with wrong credentials": {
			token: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  things.ErrUnauthorizedAccess,
		},
		"list with existing name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "chanel_name",
			},
			size: n,
			err:  nil,
		},
		"list with non-existent name": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
			},
			size: n,
			err:  nil,
		},
		"list all channels with metadata": {
			token: token,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: meta,
			},
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannels(context.Background(), tc.token, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannelsByThing(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th := ths[0]

	n := uint64(10)
	chsDisconNum := uint64(1)

	for i := uint64(0); i < n; i++ {
		schs, err := svc.CreateChannels(context.Background(), token, channel)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		sch := schs[0]

		// Don't connect last Channel
		if i == n-chsDisconNum {
			break
		}

		err = svc.Connect(context.Background(), token, []string{sch.ID}, []string{th.ID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	// Wait for things and channels to connect.
	time.Sleep(time.Second)

	cases := map[string]struct {
		token     string
		thing     string
		offset    uint64
		limit     uint64
		connected bool
		size      uint64
		err       error
	}{
		"list all channels by existing thing": {
			token:     token,
			thing:     th.ID,
			offset:    0,
			limit:     n,
			connected: true,
			size:      n - chsDisconNum,
			err:       nil,
		},
		"list half of channels by existing thing": {
			token:     token,
			thing:     th.ID,
			offset:    n / 2,
			limit:     n,
			connected: true,
			size:      (n / 2) - chsDisconNum,
			err:       nil,
		},
		"list last channel by existing thing": {
			token:     token,
			thing:     th.ID,
			offset:    n - 1 - chsDisconNum,
			limit:     n,
			connected: true,
			size:      1,
			err:       nil,
		},
		"list empty set of channels by existing thing": {
			token:     token,
			thing:     th.ID,
			offset:    n + 1,
			limit:     n,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list channels by existing thing with zero limit": {
			token:     token,
			thing:     th.ID,
			offset:    1,
			limit:     0,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list channels by existing thing with wrong credentials": {
			token:     wrongValue,
			thing:     th.ID,
			offset:    0,
			limit:     0,
			connected: true,
			size:      0,
			err:       things.ErrUnauthorizedAccess,
		},
		"list channels by non-existent thing": {
			token:     token,
			thing:     "non-existent",
			offset:    0,
			limit:     10,
			connected: true,
			size:      0,
			err:       nil,
		},
		"list all non connected channels by existing thing": {
			token:     token,
			thing:     th.ID,
			offset:    0,
			limit:     n,
			connected: false,
			size:      chsDisconNum,
			err:       nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListChannelsByThing(context.Background(), tc.token, tc.thing, tc.offset, tc.limit, tc.connected)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove channel with wrong credentials",
			id:    ch.ID,
			token: wrongValue,
			err:   things.ErrUnauthorizedAccess,
		},
		{
			desc:  "remove existing channel",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove removed channel",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing channel",
			id:    ch.ID,
			token: token,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveChannel(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

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
			chanID:  ch.ID,
			thingID: th.ID,
			err:     nil,
		},
		{
			desc:    "connect thing with wrong credentials",
			token:   wrongValue,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		{
			desc:    "connect thing to non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: th.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "connect non-existing thing to channel",
			token:   token,
			chanID:  ch.ID,
			thingID: wrongID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Connect(context.Background(), tc.token, []string{tc.chanID}, []string{tc.thingID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

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
			chanID:  ch.ID,
			thingID: th.ID,
			err:     nil,
		},
		{
			desc:    "disconnect disconnected thing",
			token:   token,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect with wrong credentials",
			token:   wrongValue,
			chanID:  ch.ID,
			thingID: th.ID,
			err:     things.ErrUnauthorizedAccess,
		},
		{
			desc:    "disconnect from non-existing channel",
			token:   token,
			chanID:  wrongID,
			thingID: th.ID,
			err:     things.ErrNotFound,
		},
		{
			desc:    "disconnect non-existing thing",
			token:   token,
			chanID:  ch.ID,
			thingID: wrongID,
			err:     things.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Disconnect(context.Background(), tc.token, tc.chanID, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccessByKey(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chs, err := svc.CreateChannels(context.Background(), token, channel, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	err = svc.Connect(context.Background(), token, []string{chs[0].ID}, []string{ths[0].ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		token   string
		channel string
		err     error
	}{
		"allowed access": {
			token:   ths[0].Key,
			channel: chs[0].ID,
			err:     nil,
		},
		"non-existing thing": {
			token:   wrongValue,
			channel: chs[0].ID,
			err:     things.ErrNotFound,
		},
		"non-existing chan": {
			token:   ths[0].Key,
			channel: wrongValue,
			err:     things.ErrEntityConnected,
		},
		"non-connected channel": {
			token:   ths[0].Key,
			channel: chs[1].ID,
			err:     things.ErrEntityConnected,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccessByKey(context.Background(), tc.channel, tc.token)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected '%s' got '%s'\n", desc, tc.err, err))
	}
}

func TestCanAccessByID(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = svc.Connect(context.Background(), token, []string{ch.ID}, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := map[string]struct {
		thingID string
		channel string
		err     error
	}{
		"allowed access": {
			thingID: th.ID,
			channel: ch.ID,
			err:     nil,
		},
		"access to non-existing thing": {
			thingID: wrongValue,
			channel: ch.ID,
			err:     things.ErrEntityConnected,
		},
		"access to non-existing channel": {
			thingID: th.ID,
			channel: wrongID,
			err:     things.ErrEntityConnected,
		},
		"access to not-connected thing": {
			thingID: ths[1].ID,
			channel: ch.ID,
			err:     things.ErrEntityConnected,
		},
	}

	for desc, tc := range cases {
		err := svc.CanAccessByID(context.Background(), tc.channel, tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService(map[string]string{token: email})

	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th := ths[0]

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify existing thing": {
			token: th.Key,
			id:    th.ID,
			err:   nil,
		},
		"identify non-existing thing": {
			token: wrongValue,
			id:    wrongID,
			err:   things.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.token)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.id, id))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
