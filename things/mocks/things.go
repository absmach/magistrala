//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/things"
)

var _ things.ThingRepository = (*thingRepositoryMock)(nil)

type thingRepositoryMock struct {
	mu      sync.Mutex
	counter uint64
	conns   chan Connection
	tconns  map[string]map[string]things.Thing
	things  map[string]things.Thing
}

// NewThingRepository creates in-memory thing repository.
func NewThingRepository(conns chan Connection) things.ThingRepository {
	repo := &thingRepositoryMock{
		conns:  conns,
		things: make(map[string]things.Thing),
		tconns: make(map[string]map[string]things.Thing),
	}
	go func(conns chan Connection, repo *thingRepositoryMock) {
		for conn := range conns {
			if !conn.connected {
				repo.disconnect(conn)
				continue
			}
			repo.connect(conn)
		}
	}(conns, repo)

	return repo
}

func (trm *thingRepositoryMock) Save(_ context.Context, thing things.Thing) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, th := range trm.things {
		if th.Key == thing.Key {
			return "", things.ErrConflict
		}
	}

	trm.counter++
	thing.ID = strconv.FormatUint(trm.counter, 10)
	trm.things[key(thing.Owner, thing.ID)] = thing

	return thing.ID, nil
}

func (trm *thingRepositoryMock) Update(_ context.Context, thing things.Thing) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	dbKey := key(thing.Owner, thing.ID)

	if _, ok := trm.things[dbKey]; !ok {
		return things.ErrNotFound
	}

	trm.things[dbKey] = thing

	return nil
}

func (trm *thingRepositoryMock) UpdateKey(_ context.Context, owner, id, val string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, th := range trm.things {
		if th.Key == val {
			return things.ErrConflict
		}
	}

	dbKey := key(owner, id)

	th, ok := trm.things[dbKey]
	if !ok {
		return things.ErrNotFound
	}

	th.Key = val
	trm.things[dbKey] = th

	return nil
}

func (trm *thingRepositoryMock) RetrieveByID(_ context.Context, owner, id string) (things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if c, ok := trm.things[key(owner, id)]; ok {
		return c, nil
	}

	return things.Thing{}, things.ErrNotFound
}

func (trm *thingRepositoryMock) RetrieveAll(_ context.Context, owner string, offset, limit uint64, name string) (things.ThingsPage, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	items := make([]things.Thing, 0)

	if offset < 0 || limit <= 0 {
		return things.ThingsPage{}, nil
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range trm.things {
		id, _ := strconv.ParseUint(v.ID, 10, 64)
		if strings.HasPrefix(k, prefix) && id >= first && id < last {
			items = append(items, v)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	page := things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  trm.counter,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (trm *thingRepositoryMock) RetrieveByChannel(_ context.Context, owner, chanID string, offset, limit uint64) (things.ThingsPage, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	items := make([]things.Thing, 0)

	if offset < 0 || limit <= 0 {
		return things.ThingsPage{}, nil
	}

	first := uint64(offset) + 1
	last := first + uint64(limit)

	ths, ok := trm.tconns[chanID]
	if !ok {
		return things.ThingsPage{}, nil
	}

	for _, v := range ths {
		id, _ := strconv.ParseUint(v.ID, 10, 64)
		if id >= first && id < last {
			items = append(items, v)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	page := things.ThingsPage{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  trm.counter,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (trm *thingRepositoryMock) Remove(_ context.Context, owner, id string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()
	delete(trm.things, key(owner, id))
	return nil
}

func (trm *thingRepositoryMock) RetrieveByKey(_ context.Context, key string) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, thing := range trm.things {
		if thing.Key == key {
			return thing.ID, nil
		}
	}

	return "", things.ErrNotFound
}

func (trm *thingRepositoryMock) connect(conn Connection) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if _, ok := trm.tconns[conn.chanID]; !ok {
		trm.tconns[conn.chanID] = make(map[string]things.Thing)
	}
	trm.tconns[conn.chanID][conn.thing.ID] = conn.thing
}

func (trm *thingRepositoryMock) disconnect(conn Connection) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if conn.thing.ID == "" {
		delete(trm.tconns, conn.chanID)
		return
	}
	delete(trm.tconns[conn.chanID], conn.thing.ID)
}

type thingCacheMock struct {
	mu     sync.Mutex
	things map[string]string
}

// NewThingCache returns mock cache instance.
func NewThingCache() things.ThingCache {
	return &thingCacheMock{
		things: make(map[string]string),
	}
}

func (tcm *thingCacheMock) Save(_ context.Context, key, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.things[key] = id
	return nil
}

func (tcm *thingCacheMock) ID(_ context.Context, key string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	id, ok := tcm.things[key]
	if !ok {
		return "", things.ErrNotFound
	}

	return id, nil
}

func (tcm *thingCacheMock) Remove(_ context.Context, id string) error {
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
