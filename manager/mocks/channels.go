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

func (repo *channelRepositoryMock) Save(channel manager.Channel) (string, error) {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	repo.counter += 1
	channel.ID = strconv.Itoa(repo.counter)

	repo.channels[key(channel.Owner, channel.ID)] = channel

	return channel.ID, nil
}

func (repo *channelRepositoryMock) Update(channel manager.Channel) error {
	repo.mu.Lock()
	defer repo.mu.Unlock()

	dbKey := key(channel.Owner, channel.ID)

	if _, ok := repo.channels[dbKey]; !ok {
		return manager.ErrNotFound
	}

	repo.channels[dbKey] = channel
	return nil
}

func (repo *channelRepositoryMock) One(owner, id string) (manager.Channel, error) {
	if c, ok := repo.channels[key(owner, id)]; ok {
		return c, nil
	}

	return manager.Channel{}, manager.ErrNotFound
}

func (repo *channelRepositoryMock) All(owner string) []manager.Channel {
	prefix := fmt.Sprintf("%s-", owner)

	channels := make([]manager.Channel, 0)

	for k, v := range repo.channels {
		if strings.HasPrefix(k, prefix) {
			channels = append(channels, v)
		}
	}

	return channels
}

func (repo *channelRepositoryMock) Remove(owner, id string) error {
	delete(repo.channels, key(owner, id))
	return nil
}

func (repo *channelRepositoryMock) HasClient(channel, client string) bool {
	suffix := fmt.Sprintf("-%s", channel)

	for k, v := range repo.channels {
		if strings.HasSuffix(k, suffix) {
			for _, c := range v.Connected {
				if c == client {
					return true
				}
			}
		}
	}

	return false
}
