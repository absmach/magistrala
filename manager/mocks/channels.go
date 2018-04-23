package mocks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/manager"
)

var _ manager.ChannelRepository = (*channelRepositoryMock)(nil)

const chanId = "123e4567-e89b-12d3-a456-"

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  int
	channels map[string]manager.Channel
	clients  manager.ClientRepository
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository(clients manager.ClientRepository) manager.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]manager.Channel),
		clients:  clients,
	}
}

func (crm *channelRepositoryMock) Save(channel manager.Channel) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.counter += 1
	channel.ID = fmt.Sprintf("%s%012d", chanId, crm.counter)

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

func (crm *channelRepositoryMock) All(owner string, offset, limit int) []manager.Channel {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	channels := make([]manager.Channel, 0)

	if offset < 0 || limit <= 0 {
		return channels
	}

	// Since IDs starts from 1, shift everything by one.
	first := fmt.Sprintf("%s%012d", chanId, offset+1)
	last := fmt.Sprintf("%s%012d", chanId, offset+limit+1)

	for k, v := range crm.channels {
		if strings.HasPrefix(k, prefix) && v.ID >= first && v.ID < last {
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

	client, err := crm.clients.One(owner, clientId)
	if err != nil {
		return err
	}
	channel.Clients = append(channel.Clients, client)
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
