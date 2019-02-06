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

var _ bootstrap.ConfigRepository = (*configRepositoryMock)(nil)

type configRepositoryMock struct {
	mu      sync.Mutex
	counter uint64
	configs map[string]bootstrap.Config
	unknown map[string]string
}

// NewConfigsRepository creates in-memory thing repository.
func NewConfigsRepository(unknown map[string]string) bootstrap.ConfigRepository {
	return &configRepositoryMock{
		configs: make(map[string]bootstrap.Config),
		unknown: unknown,
	}
}

func (crm *configRepositoryMock) Save(config bootstrap.Config) (string, error) {
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
	delete(crm.unknown, config.ExternalID)

	return config.MFThing, nil
}

func (crm *configRepositoryMock) RetrieveByID(key, id string) (bootstrap.Config, error) {
	c, ok := crm.configs[id]
	if !ok {
		return bootstrap.Config{}, bootstrap.ErrNotFound
	}
	if c.Owner != key {
		return bootstrap.Config{}, bootstrap.ErrUnauthorizedAccess
	}

	return c, nil

}

func (crm *configRepositoryMock) RetrieveAll(key string, filter bootstrap.Filter, offset, limit uint64) []bootstrap.Config {
	configs := make([]bootstrap.Config, 0)

	if offset < 0 || limit <= 0 {
		return configs
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)
	var state bootstrap.State = -1
	var name string
	if s, ok := filter.FullMatch["state"]; ok {
		val, _ := strconv.Atoi(s)
		state = bootstrap.State(val)
	}

	if s, ok := filter.PartialMatch["name"]; ok {
		name = strings.ToLower(s)
	}

	for _, v := range crm.configs {
		id, _ := strconv.ParseUint(v.MFThing, 10, 64)
		if id >= first && id < last {
			if (state == -1 || v.State == state) &&
				(name == "" || strings.Index(strings.ToLower(v.Name), name) != -1) &&
				v.Owner == key {
				configs = append(configs, v)
			}
		}
	}

	sort.SliceStable(configs, func(i, j int) bool {
		return configs[i].MFThing < configs[j].MFThing
	})

	return configs
}

func (crm *configRepositoryMock) RetrieveByExternalID(externalKey, externalID string) (bootstrap.Config, error) {
	for _, thing := range crm.configs {
		if thing.ExternalID == externalID && thing.ExternalKey == externalKey {
			return thing, nil
		}
	}

	return bootstrap.Config{}, bootstrap.ErrNotFound
}

func (crm *configRepositoryMock) Update(config bootstrap.Config) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	if _, ok := crm.configs[config.MFThing]; !ok {
		return bootstrap.ErrNotFound
	}

	crm.configs[config.MFThing] = config

	return nil
}

func (crm *configRepositoryMock) Remove(key, id string) error {
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

func (crm *configRepositoryMock) RetrieveUnknown(offset, limit uint64) []bootstrap.Config {
	res := []bootstrap.Config{}
	i := uint64(0)
	l := int(limit)
	var keys []string
	for k := range crm.unknown {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if i >= offset && len(res) < l {
			res = append(res, bootstrap.Config{
				ExternalID:  k,
				ExternalKey: crm.unknown[k],
			})
		}
		i++
	}

	return res
}

func (crm *configRepositoryMock) RemoveUnknown(key, id string) error {
	for k, v := range crm.unknown {
		if k == id && v == key {
			delete(crm.unknown, k)
			return nil
		}
	}

	return nil
}

func (crm *configRepositoryMock) SaveUnknown(key, id string) error {
	crm.unknown[id] = key
	return nil
}
