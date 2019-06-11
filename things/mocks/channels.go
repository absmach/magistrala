//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
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

func (crm *channelRepositoryMock) Save(channel things.Channel) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.counter++
	channel.ID = strconv.FormatUint(crm.counter, 10)
	crm.channels[key(channel.Owner, channel.ID)] = channel

	return channel.ID, nil
}

func (crm *channelRepositoryMock) Update(channel things.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return things.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) RetrieveByID(owner, id string) (things.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return things.Channel{}, things.ErrNotFound
}

func (crm *channelRepositoryMock) RetrieveAll(owner string, offset, limit uint64, name string) (things.ChannelsPage, error) {
	channels := make([]things.Channel, 0)

	if offset < 0 || limit <= 0 {
		return things.ChannelsPage{}, nil
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

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
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) RetrieveByThing(owner, thingID string, offset, limit uint64) (things.ChannelsPage, error) {
	channels := make([]things.Channel, 0)

	if offset < 0 || limit <= 0 {
		return things.ChannelsPage{}, nil
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

	for _, v := range crm.cconns[thingID] {
		id, _ := strconv.ParseUint(v.ID, 10, 64)
		if id >= first && id < last {
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
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) Remove(owner, id string) error {
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

func (crm *channelRepositoryMock) Connect(owner, chanID, thingID string) error {
	channel, err := crm.RetrieveByID(owner, chanID)
	if err != nil {
		return err
	}

	thing, err := crm.things.RetrieveByID(owner, thingID)
	if err != nil {
		return err
	}

	crm.tconns <- Connection{
		chanID:    chanID,
		thing:     thing,
		connected: true,
	}
	if _, ok := crm.cconns[thingID]; !ok {
		crm.cconns[thingID] = make(map[string]things.Channel)
	}
	crm.cconns[thingID][chanID] = channel
	return nil
}

func (crm *channelRepositoryMock) Disconnect(owner, chanID, thingID string) error {
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

func (crm *channelRepositoryMock) HasThing(chanID, token string) (string, error) {
	tid, err := crm.things.RetrieveByKey(token)
	if err != nil {
		return "", things.ErrNotFound
	}

	chans, ok := crm.cconns[tid]
	if !ok {
		return "", things.ErrNotFound
	}

	if _, ok := chans[chanID]; !ok {
		return "", things.ErrNotFound
	}

	return tid, nil
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

func (ccm *channelCacheMock) Connect(chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.channels[chanID] = thingID
	return nil
}

func (ccm *channelCacheMock) HasThing(chanID, thingID string) bool {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	return ccm.channels[chanID] == thingID
}

func (ccm *channelCacheMock) Disconnect(chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}

func (ccm *channelCacheMock) Remove(chanID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}
