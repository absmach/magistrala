package mocks

import (
	"fmt"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/things"
)

var _ things.ThingRepository = (*thingRepositoryMock)(nil)

const cliID = "123e4567-e89b-12d3-a456-"

type thingRepositoryMock struct {
	mu      sync.Mutex
	counter int
	things  map[string]things.Thing
}

// NewThingRepository creates in-memory thing repository.
func NewThingRepository() things.ThingRepository {
	return &thingRepositoryMock{
		things: make(map[string]things.Thing),
	}
}

func (trm *thingRepositoryMock) ID() string {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.counter++
	return fmt.Sprintf("%s%012d", cliID, trm.counter)
}

func (trm *thingRepositoryMock) Save(thing things.Thing) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.things[key(thing.Owner, thing.ID)] = thing

	return nil
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

func (trm *thingRepositoryMock) One(owner, id string) (things.Thing, error) {
	if c, ok := trm.things[key(owner, id)]; ok {
		return c, nil
	}

	return things.Thing{}, things.ErrNotFound
}

func (trm *thingRepositoryMock) All(owner string, offset, limit int) []things.Thing {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	things := make([]things.Thing, 0)

	if offset < 0 || limit <= 0 {
		return things
	}

	// Since IDs start from 1, shift everything by one.
	first := fmt.Sprintf("%s%012d", cliID, offset+1)
	last := fmt.Sprintf("%s%012d", cliID, offset+limit+1)

	for k, v := range trm.things {
		if strings.HasPrefix(k, prefix) && v.ID >= first && v.ID < last {
			things = append(things, v)
		}
	}

	return things
}

func (trm *thingRepositoryMock) Remove(owner, id string) error {
	delete(trm.things, key(owner, id))
	return nil
}
