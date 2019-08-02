//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/bootstrap"
)

const (
	emptyState  = -1
	notFoundIdx = -1
)

var _ bootstrap.ConfigRepository = (*configRepositoryMock)(nil)

type configRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	configs  map[string]bootstrap.Config
	channels map[string]bootstrap.Channel
	unknown  map[string]string
}

// NewConfigsRepository creates in-memory config repository.
func NewConfigsRepository(unknown map[string]string) bootstrap.ConfigRepository {
	return &configRepositoryMock{
		configs:  make(map[string]bootstrap.Config),
		channels: make(map[string]bootstrap.Channel),
		unknown:  unknown,
	}
}

func (crm *configRepositoryMock) Save(config bootstrap.Config, connections []string) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, v := range crm.configs {
		if v.MFThing == config.MFThing || v.ExternalID == config.ExternalID {
			return "", bootstrap.ErrConflict
		}
	}

	crm.counter++
	config.MFThing = strconv.FormatUint(crm.counter, 10)
	crm.configs[config.MFThing] = config

	for _, ch := range config.MFChannels {
		crm.channels[ch.ID] = ch
	}

	config.MFChannels = []bootstrap.Channel{}

	for _, ch := range connections {
		config.MFChannels = append(config.MFChannels, crm.channels[ch])
	}

	crm.configs[config.MFThing] = config
	delete(crm.unknown, config.ExternalID)

	return config.MFThing, nil
}

func (crm *configRepositoryMock) RetrieveByID(key, id string) (bootstrap.Config, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	c, ok := crm.configs[id]
	if !ok {
		return bootstrap.Config{}, bootstrap.ErrNotFound
	}
	if c.Owner != key {
		return bootstrap.Config{}, bootstrap.ErrUnauthorizedAccess
	}

	return c, nil

}

func (crm *configRepositoryMock) RetrieveAll(key string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	configs := make([]bootstrap.Config, 0)

	if offset < 0 || limit <= 0 {
		return bootstrap.ConfigsPage{}
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)
	var state bootstrap.State = emptyState
	var name string
	if s, ok := filter.FullMatch["state"]; ok {
		val, _ := strconv.Atoi(s)
		state = bootstrap.State(val)
	}

	if s, ok := filter.PartialMatch["name"]; ok {
		name = strings.ToLower(s)
	}

	var total uint64
	for _, v := range crm.configs {
		id, _ := strconv.ParseUint(v.MFThing, 10, 64)
		if (state == emptyState || v.State == state) &&
			(name == "" || strings.Index(strings.ToLower(v.Name), name) != notFoundIdx) &&
			v.Owner == key {
			if id >= first && id < last {
				configs = append(configs, v)
			}
			total++
		}
	}

	sort.SliceStable(configs, func(i, j int) bool {
		return configs[i].MFThing < configs[j].MFThing
	})

	return bootstrap.ConfigsPage{
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		Configs: configs,
	}
}

func (crm *configRepositoryMock) RetrieveByExternalID(externalID string) (bootstrap.Config, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, cfg := range crm.configs {
		if cfg.ExternalID == externalID {
			return cfg, nil
		}
	}

	return bootstrap.Config{}, bootstrap.ErrNotFound
}

func (crm *configRepositoryMock) Update(config bootstrap.Config) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	cfg, ok := crm.configs[config.MFThing]
	if !ok || cfg.Owner != config.Owner {
		return bootstrap.ErrNotFound
	}

	cfg.Name = config.Name
	cfg.Content = config.Content
	crm.configs[config.MFThing] = cfg

	return nil
}

func (crm *configRepositoryMock) UpdateCert(owner, thingKey, clientCert, clientKey, caCert string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()
	var forUpdate bootstrap.Config
	for _, v := range crm.configs {
		if v.MFKey == thingKey && v.Owner == owner {
			forUpdate = v
			break
		}
	}
	if _, ok := crm.configs[forUpdate.MFThing]; !ok {
		return bootstrap.ErrNotFound
	}
	forUpdate.ClientCert = clientCert
	forUpdate.ClientKey = clientKey
	forUpdate.CACert = caCert
	crm.configs[forUpdate.MFThing] = forUpdate

	return nil
}

func (crm *configRepositoryMock) UpdateConnections(key, id string, channels []bootstrap.Channel, connections []string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	config, ok := crm.configs[id]
	if !ok {
		return bootstrap.ErrNotFound
	}

	for _, ch := range channels {
		crm.channels[ch.ID] = ch
	}

	config.MFChannels = []bootstrap.Channel{}
	for _, conn := range connections {
		ch, ok := crm.channels[conn]
		if !ok {
			return bootstrap.ErrNotFound
		}
		config.MFChannels = append(config.MFChannels, ch)
	}
	crm.configs[id] = config

	return nil
}

func (crm *configRepositoryMock) Remove(key, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for k, v := range crm.configs {
		if v.Owner == key && k == id {
			delete(crm.configs, k)
			break
		}
	}

	return nil
}

func (crm *configRepositoryMock) ChangeState(key, id string, state bootstrap.State) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	config, ok := crm.configs[id]
	if !ok {
		return bootstrap.ErrNotFound
	}
	if config.Owner != key {
		return bootstrap.ErrUnauthorizedAccess
	}

	config.State = state
	crm.configs[id] = config
	return nil
}

func (crm *configRepositoryMock) RetrieveUnknown(offset, limit uint64) bootstrap.ConfigsPage {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	configs := []bootstrap.Config{}
	i := uint64(0)
	l := int(limit)
	var keys []string
	for k := range crm.unknown {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if i >= offset && len(configs) < l {
			configs = append(configs, bootstrap.Config{
				ExternalID:  k,
				ExternalKey: crm.unknown[k],
			})
		}
		i++
	}

	return bootstrap.ConfigsPage{
		Total:   uint64(len(crm.unknown)),
		Offset:  offset,
		Limit:   limit,
		Configs: configs,
	}
}

func (crm *configRepositoryMock) RemoveUnknown(key, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for k, v := range crm.unknown {
		if k == id && v == key {
			delete(crm.unknown, k)
			return nil
		}
	}

	return nil
}

func (crm *configRepositoryMock) SaveUnknown(key, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	crm.unknown[id] = key
	return nil
}

func (crm *configRepositoryMock) ListExisting(key string, connections []string) ([]bootstrap.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var ret []bootstrap.Channel

	for k, v := range crm.channels {
		for _, conn := range connections {
			if conn == k {
				ret = append(ret, v)
				break
			}
		}
	}

	return ret, nil
}

func (crm *configRepositoryMock) RemoveThing(id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	delete(crm.configs, id)
	return nil
}

func (crm *configRepositoryMock) UpdateChannel(ch bootstrap.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	channel, ok := crm.channels[ch.ID]
	if !ok {
		return nil
	}

	channel.Name = ch.Name
	channel.Metadata = ch.Metadata
	crm.channels[ch.ID] = channel
	return nil
}

func (crm *configRepositoryMock) RemoveChannel(id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	delete(crm.channels, id)
	return nil
}

func (crm *configRepositoryMock) DisconnectThing(channelID, thingID string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	idx := -1
	if config, ok := crm.configs[thingID]; ok {
		for i, ch := range config.MFChannels {
			if ch.ID == channelID {
				idx = i
				break
			}
		}

		if idx != -1 {
			config.MFChannels = append(config.MFChannels[0:idx], config.MFChannels[idx:]...)
		}
		crm.configs[thingID] = config
	}

	delete(crm.channels, channelID)

	return nil
}
