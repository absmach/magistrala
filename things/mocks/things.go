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
	mu     sync.Mutex
	things map[string]things.Thing
}

// NewThingRepository creates in-memory thing repository.
func NewThingRepository() things.ThingRepository {
	return &thingRepositoryMock{
		things: make(map[string]things.Thing),
	}
}

func (trm *thingRepositoryMock) Save(thing things.Thing) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

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

func (trm *thingRepositoryMock) RetrieveByID(owner, id string) (things.Thing, error) {
	if c, ok := trm.things[key(owner, id)]; ok {
		return c, nil
	}

	return things.Thing{}, things.ErrNotFound
}

func (trm *thingRepositoryMock) RetrieveAll(owner string, offset, limit int) []things.Thing {
	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	things := make([]things.Thing, 0)

	if offset < 0 || limit <= 0 {
		return things
	}

	// Since both ID and key are generated via the identity provider mock, all
	// identifiers will be at "odd" positions. The following loop skips all
	// values used for keys. Starting value of 1 indicates the first usable
	// UUID produced by mocked identity provider.
	skip := 1
	for i := 0; i < offset; i++ {
		skip += 2
	}

	first := fmt.Sprintf("%s%012d", startID, skip)
	last := fmt.Sprintf("%s%012d", startID, skip+2*(limit-1))

	for k, v := range trm.things {
		if strings.HasPrefix(k, prefix) && v.ID >= first && v.ID <= last {
			things = append(things, v)
		}
	}

	sort.SliceStable(things, func(i, j int) bool {
		return things[i].ID < things[j].ID
	})

	return things
}

func (trm *thingRepositoryMock) Remove(owner, id string) error {
	delete(trm.things, key(owner, id))
	return nil
}

func (trm *thingRepositoryMock) RetrieveByKey(key string) (string, error) {
	for _, thing := range trm.things {
		if thing.Key == key {
			return thing.ID, nil
		}
	}
	return "", things.ErrNotFound
}
