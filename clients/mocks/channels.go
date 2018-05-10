package mocks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/clients"
)

var _ clients.ChannelRepository = (*channelRepositoryMock)(nil)

const chanID = "123e4567-e89b-12d3-a456-"

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  int
	channels map[string]clients.Channel
	clients  clients.ClientRepository
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository(repo clients.ClientRepository) clients.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]clients.Channel),
		clients:  repo,
	}
}

func (crm *channelRepositoryMock) Save(channel clients.Channel) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.counter++
	channel.ID = fmt.Sprintf("%s%012d", chanID, crm.counter)

	crm.channels[key(channel.Owner, channel.ID)] = channel

	return channel.ID, nil
}

func (crm *channelRepositoryMock) Update(channel clients.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return clients.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) One(owner, id string) (clients.Channel, error) {
	if c, ok := crm.channels[key(owner, id)]; ok {
		return c, nil
	}

	return clients.Channel{}, clients.ErrNotFound
}

func (crm *channelRepositoryMock) All(owner string, offset, limit int) []clients.Channel {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	channels := make([]clients.Channel, 0)

	if offset < 0 || limit <= 0 {
		return channels
	}

	// Since IDs starts from 1, shift everything by one.
	first := fmt.Sprintf("%s%012d", chanID, offset+1)
	last := fmt.Sprintf("%s%012d", chanID, offset+limit+1)

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

func (crm *channelRepositoryMock) Connect(owner, chanID, clientID string) error {
	channel, err := crm.One(owner, chanID)
	if err != nil {
		return err
	}

	client, err := crm.clients.One(owner, clientID)
	if err != nil {
		return err
	}
	channel.Clients = append(channel.Clients, client)
	return crm.Update(channel)
}

func (crm *channelRepositoryMock) Disconnect(owner, chanID, clientID string) error {
	channel, err := crm.One(owner, chanID)
	if err != nil {
		return err
	}

	if !crm.HasClient(chanID, clientID) {
		return clients.ErrNotFound
	}

	connected := make([]clients.Client, len(channel.Clients)-1)
	for _, client := range channel.Clients {
		if client.ID != clientID {
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
