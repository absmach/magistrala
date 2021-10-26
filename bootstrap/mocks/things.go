// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

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
	auth        mainflux.AuthServiceClient
	connections map[string][]string
}

// NewThingsService returns Mainflux Things service mock.
// Only methods used by SDK are mocked.
func NewThingsService(things map[string]things.Thing, channels map[string]things.Channel, auth mainflux.AuthServiceClient) things.Service {
	return &mainfluxThings{
		things:      things,
		channels:    channels,
		auth:        auth,
		connections: make(map[string][]string),
	}
}

func (svc *mainfluxThings) CreateThings(_ context.Context, owner string, ths ...things.Thing) ([]things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return []things.Thing{}, things.ErrUnauthorizedAccess
	}
	for i := range ths {
		svc.counter++
		ths[i].Owner = userID.Email
		ths[i].ID = strconv.FormatUint(svc.counter, 10)
		ths[i].Key = ths[i].ID
		svc.things[ths[i].ID] = ths[i]
	}

	return ths, nil
}

func (svc *mainfluxThings) ViewThing(_ context.Context, owner, id string) (things.Thing, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.Thing{}, things.ErrUnauthorizedAccess
	}

	if t, ok := svc.things[id]; ok && t.Owner == userID.Email {
		return t, nil

	}

	return things.Thing{}, things.ErrNotFound
}

func (svc *mainfluxThings) Connect(_ context.Context, owner string, chIDs, thIDs []string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.ErrUnauthorizedAccess
	}
	for _, chID := range chIDs {
		if svc.channels[chID].Owner != userID.Email {
			return things.ErrUnauthorizedAccess
		}
		for _, thID := range thIDs {
			svc.connections[chID] = append(svc.connections[chID], thID)
		}
	}

	return nil
}

func (svc *mainfluxThings) Disconnect(_ context.Context, owner string, chIDs, thIDs []string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.ErrUnauthorizedAccess
	}

	for _, chID := range chIDs {
		if svc.channels[chID].Owner != userID.Email {
			return things.ErrUnauthorizedAccess
		}

		ids := svc.connections[chID]
		var count int
		var newConns []string
		for _, thID := range thIDs {
			for _, id := range ids {
				if id == thID {
					count++
					continue
				}
				newConns = append(newConns, id)
			}

			if len(newConns)-len(ids) != count {
				return things.ErrNotFound
			}
			svc.connections[chID] = newConns
		}
	}
	return nil
}

func (svc *mainfluxThings) RemoveThing(_ context.Context, owner, id string) error {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return things.ErrUnauthorizedAccess
	}

	if t, ok := svc.things[id]; !ok || t.Owner != userID.Email {
		return things.ErrNotFound
	}

	delete(svc.things, id)
	conns := make(map[string][]string)
	for k, v := range svc.connections {
		i := findIndex(v, id)
		if i != -1 {
			var tmp []string
			if i != len(v)-2 {
				tmp = v[i+1:]
			}
			conns[k] = append(v[:i], tmp...)
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

func (svc *mainfluxThings) ListThings(context.Context, string, things.PageMetadata) (things.Page, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListChannelsByThing(context.Context, string, string, things.PageMetadata) (things.ChannelsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListThingsByChannel(context.Context, string, string, things.PageMetadata) (things.Page, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CreateChannels(_ context.Context, owner string, chs ...things.Channel) ([]things.Channel, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(context.Background(), &mainflux.Token{Value: owner})
	if err != nil {
		return []things.Channel{}, things.ErrUnauthorizedAccess
	}
	for i := range chs {
		svc.counter++
		chs[i].Owner = userID.Email
		chs[i].ID = strconv.FormatUint(svc.counter, 10)
		svc.channels[chs[i].ID] = chs[i]
	}

	return chs, nil
}

func (svc *mainfluxThings) UpdateChannel(context.Context, string, things.Channel) error {
	panic("not implemented")
}

func (svc *mainfluxThings) ListChannels(context.Context, string, things.PageMetadata) (things.ChannelsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) RemoveChannel(context.Context, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) CanAccessByKey(context.Context, string, string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) CanAccessByID(context.Context, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) IsChannelOwner(context.Context, string, string) error {
	panic("not implemented")
}

func (svc *mainfluxThings) Identify(context.Context, string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ShareThing(ctx context.Context, token, thingID string, actions, userIDs []string) error {
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

func (svc *mainfluxThings) ListMembers(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.Page, error) {
	panic("not implemented")
}
