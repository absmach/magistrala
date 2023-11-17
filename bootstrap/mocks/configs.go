// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/absmach/magistrala/bootstrap"
	"github.com/absmach/magistrala/pkg/errors"
)

const emptyState = -1

var _ bootstrap.ConfigRepository = (*configRepositoryMock)(nil)

type configRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	configs  map[string]bootstrap.Config
	channels map[string]bootstrap.Channel
}

// NewConfigsRepository creates in-memory config repository.
func NewConfigsRepository() bootstrap.ConfigRepository {
	return &configRepositoryMock{
		configs:  make(map[string]bootstrap.Config),
		channels: make(map[string]bootstrap.Channel),
	}
}

func (crm *configRepositoryMock) Save(_ context.Context, config bootstrap.Config, connections []string) (string, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, v := range crm.configs {
		if v.ThingID == config.ThingID || v.ExternalID == config.ExternalID {
			return "", errors.ErrConflict
		}
	}

	crm.counter++
	config.ThingID = strconv.FormatUint(crm.counter, 10)
	crm.configs[config.ThingID] = config

	for _, ch := range config.Channels {
		crm.channels[ch.ID] = ch
	}

	config.Channels = []bootstrap.Channel{}

	for _, ch := range connections {
		config.Channels = append(config.Channels, crm.channels[ch])
	}

	crm.configs[config.ThingID] = config

	return config.ThingID, nil
}

func (crm *configRepositoryMock) RetrieveByID(_ context.Context, token, id string) (bootstrap.Config, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	c, ok := crm.configs[id]
	if !ok {
		return bootstrap.Config{}, errors.ErrNotFound
	}
	if c.Owner != token {
		return bootstrap.Config{}, errors.ErrAuthentication
	}

	return c, nil
}

func (crm *configRepositoryMock) RetrieveAll(_ context.Context, token string, filter bootstrap.Filter, offset, limit uint64) bootstrap.ConfigsPage {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	configs := make([]bootstrap.Config, 0)

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
		id, _ := strconv.ParseUint(v.ThingID, 10, 64)
		if (state == emptyState || v.State == state) &&
			(name == "" || strings.Contains(strings.ToLower(v.Name), name)) &&
			v.Owner == token {
			if id >= first && id < last {
				configs = append(configs, v)
			}
			total++
		}
	}

	sort.SliceStable(configs, func(i, j int) bool {
		return configs[i].ThingID < configs[j].ThingID
	})

	return bootstrap.ConfigsPage{
		Total:   total,
		Offset:  offset,
		Limit:   limit,
		Configs: configs,
	}
}

func (crm *configRepositoryMock) RetrieveByExternalID(_ context.Context, externalID string) (bootstrap.Config, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, cfg := range crm.configs {
		if cfg.ExternalID == externalID {
			return cfg, nil
		}
	}

	return bootstrap.Config{}, errors.ErrNotFound
}

func (crm *configRepositoryMock) Update(_ context.Context, config bootstrap.Config) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	cfg, ok := crm.configs[config.ThingID]
	if !ok || cfg.Owner != config.Owner {
		return errors.ErrNotFound
	}

	cfg.Name = config.Name
	cfg.Content = config.Content
	crm.configs[config.ThingID] = cfg

	return nil
}

func (crm *configRepositoryMock) UpdateCert(_ context.Context, owner, thingID, clientCert, clientKey, caCert string) (bootstrap.Config, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()
	var forUpdate bootstrap.Config
	for _, v := range crm.configs {
		if v.ThingID == thingID && v.Owner == owner {
			forUpdate = v
			break
		}
	}
	if _, ok := crm.configs[forUpdate.ThingID]; !ok {
		return bootstrap.Config{}, errors.ErrNotFound
	}
	forUpdate.ClientCert = clientCert
	forUpdate.ClientKey = clientKey
	forUpdate.CACert = caCert
	crm.configs[forUpdate.ThingID] = forUpdate

	return forUpdate, nil
}

func (crm *configRepositoryMock) UpdateConnections(_ context.Context, token, id string, channels []bootstrap.Channel, connections []string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	config, ok := crm.configs[id]
	if !ok {
		return errors.ErrNotFound
	}

	for _, ch := range channels {
		crm.channels[ch.ID] = ch
	}

	config.Channels = []bootstrap.Channel{}
	for _, conn := range connections {
		ch, ok := crm.channels[conn]
		if !ok {
			return errors.ErrNotFound
		}
		config.Channels = append(config.Channels, ch)
	}
	crm.configs[id] = config

	return nil
}

func (crm *configRepositoryMock) Remove(_ context.Context, token, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for k, v := range crm.configs {
		if v.Owner == token && k == id {
			delete(crm.configs, k)
			break
		}
	}

	return nil
}

func (crm *configRepositoryMock) ChangeState(_ context.Context, token, id string, state bootstrap.State) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	config, ok := crm.configs[id]
	if !ok {
		return errors.ErrNotFound
	}
	if config.Owner != token {
		return errors.ErrAuthentication
	}

	config.State = state
	crm.configs[id] = config
	return nil
}

func (crm *configRepositoryMock) ListExisting(_ context.Context, token string, connections []string) ([]bootstrap.Channel, error) {
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

func (crm *configRepositoryMock) RemoveThing(_ context.Context, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	delete(crm.configs, id)
	return nil
}

func (crm *configRepositoryMock) UpdateChannel(_ context.Context, ch bootstrap.Channel) error {
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

func (crm *configRepositoryMock) RemoveChannel(_ context.Context, id string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	delete(crm.channels, id)
	return nil
}

func (crm *configRepositoryMock) DisconnectThing(_ context.Context, channelID, thingID string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	idx := -1
	if config, ok := crm.configs[thingID]; ok {
		for i, ch := range config.Channels {
			if ch.ID == channelID {
				idx = i
				break
			}
		}

		if idx != -1 {
			config.Channels = append(config.Channels[0:idx], config.Channels[idx:]...)
		}
		crm.configs[thingID] = config
	}

	delete(crm.channels, channelID)

	return nil
}
