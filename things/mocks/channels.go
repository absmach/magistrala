// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/things"
)

// Connection represents connection between channel and thing that is used for
// testing purposes.
type Connection struct {
	chanID    string
	thing     things.Thing
	connected bool
}

var _ things.ChannelRepository = (*channelRepositoryMock)(nil)

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	channels map[string]things.Channel
	tconns   chan Connection                      // used for syncronization with thing repo
	cconns   map[string]map[string]things.Channel // used to track connections
	things   things.ThingRepository
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository(repo things.ThingRepository, tconns chan Connection) things.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]things.Channel),
		tconns:   tconns,
		cconns:   make(map[string]map[string]things.Channel),
		things:   repo,
	}
}

func (crm *channelRepositoryMock) Save(_ context.Context, channels ...things.Channel) ([]things.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for i := range channels {
		crm.counter++
		channels[i].ID = strconv.FormatUint(crm.counter, 10)
		crm.channels[key(channels[i].Owner, channels[i].ID)] = channels[i]
	}

	return channels, nil
}

func (crm *channelRepositoryMock) Update(_ context.Context, channel things.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return things.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) RetrieveByID(_ context.Context, owner, id string) (things.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return things.Channel{}, things.ErrNotFound
}

func (crm *channelRepositoryMock) RetrieveAll(_ context.Context, owner string, pm things.PageMetadata) (things.ChannelsPage, error) {
	channels := make([]things.Channel, 0)

	if pm.Offset < 0 || pm.Limit <= 0 {
		return things.ChannelsPage{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range crm.channels {
		id, _ := strconv.ParseUint(v.ID, 10, 64)
		if strings.HasPrefix(k, prefix) && id >= first && id < last {
			channels = append(channels, v)
		}
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return channels[i].ID < channels[j].ID
	})

	page := things.ChannelsPage{
		Channels: channels,
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) RetrieveByThing(_ context.Context, owner, thingID string, offset, limit uint64, connected bool) (things.ChannelsPage, error) {
	channels := make([]things.Channel, 0)

	if offset < 0 || limit <= 0 {
		return things.ChannelsPage{}, nil
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

	// Append connected or not connected channels
	switch connected {
	case true:
		for _, co := range crm.cconns[thingID] {
			id, _ := strconv.ParseUint(co.ID, 10, 64)
			if id >= first && id < last {
				channels = append(channels, co)
			}
		}
	default:
		for _, ch := range crm.channels {
			conn := false
			id, _ := strconv.ParseUint(ch.ID, 10, 64)
			if id >= first && id < last {
				for _, co := range crm.cconns[thingID] {
					if ch.ID == co.ID {
						conn = true
					}
				}

				// Append if not found in connections list
				if !conn {
					channels = append(channels, ch)
				}
			}
		}
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return channels[i].ID < channels[j].ID
	})

	page := things.ChannelsPage{
		Channels: channels,
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) Remove(_ context.Context, owner, id string) error {
	delete(crm.channels, key(owner, id))
	// delete channel from any thing list
	for thk := range crm.cconns {
		delete(crm.cconns[thk], key(owner, id))
	}
	crm.tconns <- Connection{
		chanID:    id,
		connected: false,
	}
	return nil
}

func (crm *channelRepositoryMock) Connect(_ context.Context, owner string, chIDs, thIDs []string) error {
	for _, chID := range chIDs {
		ch, err := crm.RetrieveByID(context.Background(), owner, chID)
		if err != nil {
			return err
		}

		for _, thID := range thIDs {
			th, err := crm.things.RetrieveByID(context.Background(), owner, thID)
			if err != nil {
				return err
			}

			crm.tconns <- Connection{
				chanID:    chID,
				thing:     th,
				connected: true,
			}
			if _, ok := crm.cconns[thID]; !ok {
				crm.cconns[thID] = make(map[string]things.Channel)
			}
			crm.cconns[thID][chID] = ch
		}
	}

	return nil
}

func (crm *channelRepositoryMock) Disconnect(_ context.Context, owner, chanID, thingID string) error {
	if _, ok := crm.cconns[thingID]; !ok {
		return things.ErrNotFound
	}

	if _, ok := crm.cconns[thingID][chanID]; !ok {
		return things.ErrNotFound
	}

	crm.tconns <- Connection{
		chanID:    chanID,
		thing:     things.Thing{ID: thingID, Owner: owner},
		connected: false,
	}
	delete(crm.cconns[thingID], chanID)
	return nil
}

func (crm *channelRepositoryMock) HasThing(_ context.Context, chanID, token string) (string, error) {
	tid, err := crm.things.RetrieveByKey(context.Background(), token)
	if err != nil {
		return "", err
	}

	chans, ok := crm.cconns[tid]
	if !ok {
		return "", things.ErrEntityConnected
	}

	if _, ok := chans[chanID]; !ok {
		return "", things.ErrEntityConnected
	}

	return tid, nil
}

func (crm *channelRepositoryMock) HasThingByID(_ context.Context, chanID, thingID string) error {
	chans, ok := crm.cconns[thingID]
	if !ok {
		return things.ErrEntityConnected
	}

	if _, ok := chans[chanID]; !ok {
		return things.ErrEntityConnected
	}

	return nil
}

type channelCacheMock struct {
	mu       sync.Mutex
	channels map[string]string
}

// NewChannelCache returns mock cache instance.
func NewChannelCache() things.ChannelCache {
	return &channelCacheMock{
		channels: make(map[string]string),
	}
}

func (ccm *channelCacheMock) Connect(_ context.Context, chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.channels[chanID] = thingID
	return nil
}

func (ccm *channelCacheMock) HasThing(_ context.Context, chanID, thingID string) bool {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	return ccm.channels[chanID] == thingID
}

func (ccm *channelCacheMock) Disconnect(_ context.Context, chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}

func (ccm *channelCacheMock) Remove(_ context.Context, chanID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}
