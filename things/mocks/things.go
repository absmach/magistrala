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
	"strings"
	"sync"

	"github.com/mainflux/mainflux/things"
)

var _ things.ThingRepository = (*thingRepositoryMock)(nil)

type thingRepositoryMock struct {
	mu      sync.Mutex
	counter uint64
	things  map[string]things.Thing
}

// NewThingRepository creates in-memory thing repository.
func NewThingRepository() things.ThingRepository {
	return &thingRepositoryMock{
		things: make(map[string]things.Thing),
	}
}

func (trm *thingRepositoryMock) Save(thing things.Thing) (uint64, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.counter++
	thing.ID = trm.counter
	trm.things[key(thing.Owner, thing.ID)] = thing

	return thing.ID, nil
}

func (trm *thingRepositoryMock) Update(thing things.Thing) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	dbKey := key(thing.Owner, thing.ID)

	if _, ok := trm.things[dbKey]; !ok {
		return things.ErrNotFound
	}

	trm.things[dbKey] = thing

	return nil
}

func (trm *thingRepositoryMock) RetrieveByID(owner string, id uint64) (things.Thing, error) {
	if c, ok := trm.things[key(owner, id)]; ok {
		return c, nil
	}

	return things.Thing{}, things.ErrNotFound
}

func (trm *thingRepositoryMock) RetrieveAll(owner string, offset, limit int) []things.Thing {
	things := make([]things.Thing, 0)

	if offset < 0 || limit <= 0 {
		return things
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range trm.things {
		if strings.HasPrefix(k, prefix) && v.ID >= first && v.ID < last {
			things = append(things, v)
		}
	}

	sort.SliceStable(things, func(i, j int) bool {
		return things[i].ID < things[j].ID
	})

	return things
}

func (trm *thingRepositoryMock) Remove(owner string, id uint64) error {
	delete(trm.things, key(owner, id))
	return nil
}

func (trm *thingRepositoryMock) RetrieveByKey(key string) (uint64, error) {
	for _, thing := range trm.things {
		if thing.Key == key {
			return thing.ID, nil
		}
	}
	return 0, things.ErrNotFound
}

type thingCacheMock struct {
	mu     sync.Mutex
	things map[string]uint64
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		things: make(map[string]uint64),
	}
}

func (tcm *thingCacheMock) Save(key string, id uint64) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.things[key] = id
	return nil
}

func (tcm *thingCacheMock) ID(key string) (uint64, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	id, ok := tcm.things[key]
	if !ok {
		return 0, things.ErrNotFound
	}

	return id, nil
}

func (tcm *thingCacheMock) Remove(id uint64) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	for key, val := range tcm.things {
		if val == id {
			delete(tcm.things, key)
			return nil
		}
	}

	return things.ErrNotFound
}
