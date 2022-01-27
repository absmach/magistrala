// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/pkg/errors"
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
	tconns   chan Connection                      // used for synchronization with thing repo
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
		if channels[i].ID == "" {
			channels[i].ID = fmt.Sprintf("%03d", crm.counter)
		}
		crm.channels[key(channels[i].Owner, channels[i].ID)] = channels[i]
	}

	return channels, nil
}

func (crm *channelRepositoryMock) Update(_ context.Context, channel things.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return errors.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) RetrieveByID(_ context.Context, owner, id string) (things.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return things.Channel{}, errors.ErrNotFound
}

func (crm *channelRepositoryMock) RetrieveAll(_ context.Context, owner string, pm things.PageMetadata) (things.ChannelsPage, error) {
	if pm.Limit < 0 {
		return things.ChannelsPage{}, nil
	}
	if pm.Limit == 0 {
		pm.Limit = 10
	}

	first := int(pm.Offset)
	last := first + int(pm.Limit)

	var chs []things.Channel

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range crm.channels {
		if strings.HasPrefix(k, prefix) {
			chs = append(chs, v)
		}
	}

	// Sort Channels list
	chs = sortChannels(pm, chs)

	if last > len(chs) {
		last = len(chs)
	}

	if first > last {
		return things.ChannelsPage{}, nil
	}

	page := things.ChannelsPage{
		Channels: chs[first:last],
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) RetrieveByThing(_ context.Context, owner, thID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	if pm.Limit <= 0 {
		return things.ChannelsPage{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	var chs []things.Channel

	// Append connected or not connected channels
	switch pm.Disconnected {
	case false:
		for _, co := range crm.cconns[thID] {
			id := parseID(co.ID)
			if id >= first && id < last {
				chs = append(chs, co)
			}
		}
	default:
		for _, ch := range crm.channels {
			conn := false
			id := parseID(ch.ID)
			if id >= first && id < last {
				for _, co := range crm.cconns[thID] {
					if ch.ID == co.ID {
						conn = true
					}
				}

				// Append if not found in connections list
				if !conn {
					chs = append(chs, ch)
				}
			}
		}
	}

	// Sort Channels by Thing list
	chs = sortChannels(pm, chs)

	page := things.ChannelsPage{
		Channels: chs,
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
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

func (crm *channelRepositoryMock) Disconnect(_ context.Context, owner string, chIDs, thIDs []string) error {
	for _, chID := range chIDs {
		for _, thID := range thIDs {
			if _, ok := crm.cconns[thID]; !ok {
				return errors.ErrNotFound
			}

			if _, ok := crm.cconns[thID][chID]; !ok {
				return errors.ErrNotFound
			}

			crm.tconns <- Connection{
				chanID:    chID,
				thing:     things.Thing{ID: thID, Owner: owner},
				connected: false,
			}
			delete(crm.cconns[thID], chID)
		}
	}

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
