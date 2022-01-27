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

func (trm *thingRepositoryMock) Save(_ context.Context, ths ...things.Thing) ([]things.Thing, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for i := range ths {
		for _, th := range trm.things {
			if th.Key == ths[i].Key {
				return []things.Thing{}, errors.ErrConflict
			}
		}

		trm.counter++
		if ths[i].ID == "" {
			ths[i].ID = fmt.Sprintf("%03d", trm.counter)
		}
		trm.things[key(ths[i].Owner, ths[i].ID)] = ths[i]
	}

	return ths, nil
}

func (trm *thingRepositoryMock) Update(_ context.Context, thing things.Thing) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	dbKey := key(thing.Owner, thing.ID)

	if _, ok := trm.things[dbKey]; !ok {
		return errors.ErrNotFound
	}

	trm.things[dbKey] = thing

	return nil
}

func (trm *thingRepositoryMock) UpdateKey(_ context.Context, owner, id, val string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, th := range trm.things {
		if th.Key == val {
			return errors.ErrConflict
		}
	}

	dbKey := key(owner, id)

	th, ok := trm.things[dbKey]
	if !ok {
		return errors.ErrNotFound
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

	return things.Thing{}, errors.ErrNotFound
}

func (trm *thingRepositoryMock) RetrieveAll(_ context.Context, owner string, pm things.PageMetadata) (things.Page, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if pm.Limit < 0 {
		return things.Page{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	var ths []things.Thing

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range trm.things {
		id := parseID(v.ID)
		if strings.HasPrefix(k, prefix) && id >= first && id < last {
			ths = append(ths, v)
		}
	}

	// Sort Things list
	ths = sortThings(pm, ths)

	page := things.Page{
		Things: ths,
		PageMetadata: things.PageMetadata{
			Total:  trm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (trm *thingRepositoryMock) RetrieveByIDs(_ context.Context, thingIDs []string, pm things.PageMetadata) (things.Page, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	items := make([]things.Thing, 0)

	if pm.Limit == 0 {
		return things.Page{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	for _, id := range thingIDs {
		suffix := fmt.Sprintf("-%s", id)
		for k, v := range trm.things {
			id := parseID(v.ID)
			if strings.HasSuffix(k, suffix) && id >= first && id < last {
				items = append(items, v)
			}
		}
	}

	items = sortThings(pm, items)

	page := things.Page{
		Things: items,
		PageMetadata: things.PageMetadata{
			Total:  trm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (trm *thingRepositoryMock) RetrieveByChannel(_ context.Context, owner, chID string, pm things.PageMetadata) (things.Page, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	if pm.Limit <= 0 {
		return things.Page{}, nil
	}

	first := uint64(pm.Offset) + 1
	last := first + uint64(pm.Limit)

	var ths []things.Thing

	// Append connected or not connected channels
	switch pm.Disconnected {
	case false:
		for _, co := range trm.tconns[chID] {
			id := parseID(co.ID)
			if id >= first && id < last {
				ths = append(ths, co)
			}
		}
	default:
		for _, th := range trm.things {
			conn := false
			id := parseID(th.ID)
			if id >= first && id < last {
				for _, co := range trm.tconns[chID] {
					if th.ID == co.ID {
						conn = true
					}
				}

				// Append if not found in connections list
				if !conn {
					ths = append(ths, th)
				}
			}
		}
	}

	// Sort Things by Channel list
	ths = sortThings(pm, ths)

	page := things.Page{
		Things: ths,
		PageMetadata: things.PageMetadata{
			Total:  trm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
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

	return "", errors.ErrNotFound
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
		return "", errors.ErrNotFound
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

	return nil
}
