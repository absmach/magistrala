//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package things_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	wrongID    = 0
	wrongValue = "wrong-value"
	email      = "user@example.com"
	token      = "token"
)

var (
	thing   = things.Thing{Type: "app", Name: "test"}
	channel = things.Channel{Name: "test", Things: []things.Thing{}}
)

func newService(tokens map[string]string) things.Service {
	users := mocks.NewUsersService(tokens)
	thingsRepo := mocks.NewThingRepository()
	channelsRepo := mocks.NewChannelRepository(thingsRepo)
	idp := mocks.NewIdentityProvider()

	return things.New(users, thingsRepo, channelsRepo, idp)
}

func TestAddThing(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"add new app":                      {thing: things.Thing{Type: "app", Name: "a"}, key: token, err: nil},
		"add new device":                   {thing: things.Thing{Type: "device", Name: "b"}, key: token, err: nil},
		"add thing with wrong credentials": {thing: things.Thing{Type: "app", Name: "d"}, key: wrongValue, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.AddThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)
	other := things.Thing{ID: wrongID, Type: "app", Key: "x"}

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"update existing thing":               {thing: saved, key: token, err: nil},
		"update thing with wrong credentials": {thing: saved, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"update non-existing thing":           {thing: other, key: token, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		id  uint64
		key string
		err error
	}{
		"view existing thing":               {id: saved.ID, key: token, err: nil},
		"view thing with wrong credentials": {id: saved.ID, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"view non-existing thing":           {id: wrongID, key: token, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewThing(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.AddThing(token, thing)
	}

	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list all things":             {key: token, offset: 0, limit: n, size: n, err: nil},
		"list half":                   {key: token, offset: n / 2, limit: n, size: n / 2, err: nil},
		"list last thing":             {key: token, offset: n - 1, limit: n, size: 1, err: nil},
		"list empty set":              {key: token, offset: n + 1, limit: n, size: 0, err: nil},
		"list with negative offset":   {key: token, offset: -1, limit: n, size: 0, err: nil},
		"list with negative limit":    {key: token, offset: 1, limit: -n, size: 0, err: nil},
		"list with zero limit":        {key: token, offset: 1, limit: 0, size: 0, err: nil},
		"list with wrong credentials": {key: wrongValue, offset: 0, limit: 0, size: 0, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		ts, err := svc.ListThings(tc.key, tc.offset, tc.limit)
		size := len(ts)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		id  uint64
		key string
		err error
	}{
		"remove thing with wrong credentials": {id: saved.ID, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"remove existing thing":               {id: saved.ID, key: token, err: nil},
		"remove removed thing":                {id: saved.ID, key: token, err: nil},
		"remove non-existing thing":           {id: wrongID, key: token, err: nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveThing(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"create channel":                        {channel: channel, key: token, err: nil},
		"create channel with wrong credentials": {channel: channel, key: wrongValue, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)
	other := things.Channel{ID: wrongID}

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"update existing channel":               {channel: saved, key: token, err: nil},
		"update channel with wrong credentials": {channel: saved, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"update non-existing channel":           {channel: other, key: token, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		id  uint64
		key string
		err error
	}{
		"view existing channel":               {id: saved.ID, key: token, err: nil},
		"view channel with wrong credentials": {id: saved.ID, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"view non-existing channel":           {id: wrongID, key: token, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.CreateChannel(token, channel)
	}
	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list all channels":           {key: token, offset: 0, limit: n, size: n, err: nil},
		"list half":                   {key: token, offset: n / 2, limit: n, size: n / 2, err: nil},
		"list last channel":           {key: token, offset: n - 1, limit: n, size: 1, err: nil},
		"list empty set":              {key: token, offset: n + 1, limit: n, size: 0, err: nil},
		"list with negative offset":   {key: token, offset: -1, limit: n, size: 0, err: nil},
		"list with negative limit":    {key: token, offset: 1, limit: -n, size: 0, err: nil},
		"list with zero limit":        {key: token, offset: 1, limit: 0, size: 0, err: nil},
		"list with wrong credentials": {key: wrongValue, offset: 0, limit: 0, size: 0, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		ch, err := svc.ListChannels(tc.key, tc.offset, tc.limit)
		size := len(ch)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		id  uint64
		key string
		err error
	}{
		"remove channel with wrong credentials": {id: saved.ID, key: wrongValue, err: things.ErrUnauthorizedAccess},
		"remove existing channel":               {id: saved.ID, key: token, err: nil},
		"remove removed channel":                {id: saved.ID, key: token, err: nil},
		"remove non-existing channel":           {id: saved.ID, key: token, err: nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		key     string
		chanID  uint64
		thingID uint64
		err     error
	}{
		"connect thing":                         {key: token, chanID: sch.ID, thingID: sth.ID, err: nil},
		"connect thing with wrong credentials":  {key: wrongValue, chanID: sch.ID, thingID: sth.ID, err: things.ErrUnauthorizedAccess},
		"connect thing to non-existing channel": {key: token, chanID: wrongID, thingID: sth.ID, err: things.ErrNotFound},
		"connect non-existing thing to channel": {key: token, chanID: sch.ID, thingID: wrongID, err: things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.Connect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, sch.ID, sth.ID)

	cases := []struct {
		desc    string
		key     string
		chanID  uint64
		thingID uint64
		err     error
	}{
		{desc: "disconnect connected thing", key: token, chanID: sch.ID, thingID: sth.ID, err: nil},
		{desc: "disconnect disconnected thing", key: token, chanID: sch.ID, thingID: sth.ID, err: things.ErrNotFound},
		{desc: "disconnect with wrong credentials", key: wrongValue, chanID: sch.ID, thingID: sth.ID, err: things.ErrUnauthorizedAccess},
		{desc: "disconnect from non-existing channel", key: token, chanID: wrongID, thingID: sth.ID, err: things.ErrNotFound},
		{desc: "disconnect non-existing thing", key: token, chanID: sch.ID, thingID: wrongID, err: things.ErrNotFound},
	}

	for _, tc := range cases {
		err := svc.Disconnect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccess(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, sch.ID, sth.ID)

	cases := map[string]struct {
		key     string
		channel uint64
		err     error
	}{
		"allowed access":                 {key: sth.Key, channel: sch.ID, err: nil},
		"not-connected cannot access":    {key: wrongValue, channel: sch.ID, err: things.ErrUnauthorizedAccess},
		"access to non-existing channel": {key: sth.Key, channel: wrongID, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccess(tc.channel, tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		key string
		id  uint64
		err error
	}{
		"identify existing thing":     {key: sth.Key, id: sth.ID, err: nil},
		"identify non-existing thing": {key: wrongValue, id: wrongID, err: things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(tc.key)
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.id, id))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
