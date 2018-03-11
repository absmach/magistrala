package mocks

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/manager"
)

var _ manager.ChannelRepository = (*channelRepositoryMock)(nil)

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  int
	channels map[string]manager.Channel
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository() manager.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]manager.Channel),
	}
}

func (crm *channelRepositoryMock) Save(channel manager.Channel) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.counter += 1
	channel.ID = strconv.Itoa(crm.counter)

	crm.channels[key(channel.Owner, channel.ID)] = channel

	return channel.ID, nil
}

func (crm *channelRepositoryMock) Update(channel manager.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return manager.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) One(owner, id string) (manager.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return manager.Channel{}, manager.ErrNotFound
}

func (crm *channelRepositoryMock) All(owner string) []manager.Channel {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)

	channels := make([]manager.Channel, 0)

	for k, v := range crm.channels {
		if strings.HasPrefix(k, prefix) {
			channels = append(channels, v)
		}
	}

	return channels
}

func (crm *channelRepositoryMock) Remove(owner, id string) error {
	delete(crm.channels, key(owner, id))
	return nil
}

func (crm *channelRepositoryMock) Connect(owner, chanId, clientId string) error {
	channel, err := crm.One(owner, chanId)
	if err != nil {
		return err
	}

	// Since the current implementation has no way to retrieve a real client
	// instance, the implementation will assume client always exist and create
	// a dummy one, containing only the provided ID.
	channel.Clients = append(channel.Clients, manager.Client{ID: clientId})
	return crm.Update(channel)
}

func (crm *channelRepositoryMock) Disconnect(owner, chanId, clientId string) error {
	channel, err := crm.One(owner, chanId)
	if err != nil {
		return err
	}

	if !crm.HasClient(chanId, clientId) {
		return manager.ErrNotFound
	}

	connected := make([]manager.Client, len(channel.Clients)-1)
	for _, client := range channel.Clients {
		if client.ID != clientId {
			connected = append(connected, client)
		}
	}

	channel.Clients = connected
	return crm.Update(channel)
}

func (crm *channelRepositoryMock) HasClient(channel, client string) bool {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	suffix := fmt.Sprintf("-%s", channel)

	for k, v := range crm.channels {
		if strings.HasSuffix(k, suffix) {
			for _, c := range v.Clients {
				if c.ID == client {
					return true
				}
			}
			break
		}
	}

	return false
}
