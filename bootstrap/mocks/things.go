//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"
	"strconv"
	"sync"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/things"
)

var _ things.Service = (*mainfluxThings)(nil)

type mainfluxThings struct {
	mu          sync.Mutex
	counter     uint64
	things      map[string]things.Thing
	channels    map[string]things.Channel
	users       mainflux.UsersServiceClient
	connections map[string][]string
}

// NewThingsService returns Mainflux Things service mock.
// Only methods used by SDK are mocked.
func NewThingsService(things map[string]things.Thing, channels map[string]things.Channel, users mainflux.UsersServiceClient) things.Service {
	return &mainfluxThings{
		things:      things,
		channels:    channels,
		users:       users,
		connections: make(map[string][]string),
	}
}

func (svc *mainfluxThings) AddThing(_ context.Context, owner string, thing things.Thing) (things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.users.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.Thing{}, things.ErrUnauthorizedAccess
	}

	svc.counter++
	thing.Owner = userID.Value
	thing.ID = strconv.FormatUint(svc.counter, 10)
	thing.Key = thing.ID
	svc.things[thing.ID] = thing
	return thing, nil
}

func (svc *mainfluxThings) ViewThing(_ context.Context, owner, id string) (things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.users.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.Thing{}, things.ErrUnauthorizedAccess
	}

	if t, ok := svc.things[id]; ok && t.Owner == userID.Value {
		return t, nil

	}

	return things.Thing{}, things.ErrNotFound
}

func (svc *mainfluxThings) Connect(_ context.Context, owner, chanID, thingID string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.users.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.ErrUnauthorizedAccess
	}

	if svc.channels[chanID].Owner != userID.Value {
		return things.ErrNotFound
	}

	svc.connections[chanID] = append(svc.connections[chanID], thingID)
	return nil
}

func (svc *mainfluxThings) Disconnect(_ context.Context, owner, chanID, thingID string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.users.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil || svc.channels[chanID].Owner != userID.Value {
		return things.ErrUnauthorizedAccess
	}

	ids := svc.connections[chanID]
	i := 0
	for _, t := range ids {
		if t == thingID {
			break
		}
		i++
	}

	if i == len(ids) {
		return things.ErrNotFound
	}

	var tmp []string
	if i != len(ids)-2 {
		tmp = ids[i+1:]
	}
	ids = append(ids[:i], tmp...)
	svc.connections[chanID] = ids

	return nil
}

func (svc *mainfluxThings) RemoveThing(_ context.Context, owner, id string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.users.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.ErrUnauthorizedAccess
	}

	if t, ok := svc.things[id]; !ok || t.Owner != userID.Value {
		return things.ErrNotFound
	}

	delete(svc.things, id)
	conns := make(map[string][]string)
	for k, v := range svc.connections {
		idx := findIndex(v, id)
		if idx != -1 {
			var tmp []string
			if idx != len(v)-2 {
				tmp = v[idx+1:]
			}
			conns[k] = append(v[:idx], tmp...)
		}
	}

	svc.connections = conns
	return nil
}

func (svc *mainfluxThings) ViewChannel(_ context.Context, owner, id string) (things.Channel, error) {
	if c, ok := svc.channels[id]; ok {
		return c, nil
	}
	return things.Channel{}, things.ErrNotFound
}

func (svc *mainfluxThings) UpdateThing(context.Context, string, things.Thing) error {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateKey(context.Context, string, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThings(context.Context, string, uint64, uint64, string) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListChannelsByThing(context.Context, string, string, uint64, uint64) (things.ChannelsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByChannel(context.Context, string, string, uint64, uint64) (things.ThingsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateChannel(context.Context, string, things.Channel) (things.Channel, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateChannel(context.Context, string, things.Channel) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListChannels(context.Context, string, uint64, uint64, string) (things.ChannelsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveChannel(context.Context, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CanAccess(context.Context, string, string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CanAccessByID(context.Context, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) Identify(context.Context, string) (string, error) {
	panic("not implemented")
}

func findIndex(list []string, val string) int {
	for i, v := range list {
		if v == val {
			return i
		}
	}

	return -1
}
