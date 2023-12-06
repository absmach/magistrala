// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/absmach/magistrala/twins"
)

var _ twins.StateRepository = (*stateRepositoryMock)(nil)

type stateRepositoryMock struct {
	mu     sync.Mutex
	states map[string]twins.State
}

// NewStateRepository creates in-memory twin repository.
func NewStateRepository() twins.StateRepository {
	return &stateRepositoryMock{
		states: make(map[string]twins.State),
	}
}

// SaveState persists the state.
func (srm *stateRepositoryMock) Save(ctx context.Context, st twins.State) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	srm.states[key(st.TwinID, strconv.FormatInt(st.ID, 10))] = st

	return nil
}

// UpdateState updates the state.
func (srm *stateRepositoryMock) Update(ctx context.Context, st twins.State) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	srm.states[key(st.TwinID, strconv.FormatInt(st.ID, 10))] = st

	return nil
}

// CountStates returns the number of states related to twin.
func (srm *stateRepositoryMock) Count(ctx context.Context, tw twins.Twin) (int64, error) {
	return int64(len(srm.states)), nil
}

func (srm *stateRepositoryMock) RetrieveAll(ctx context.Context, offset, limit uint64, twinID string) (twins.StatesPage, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	if limit <= 0 {
		return twins.StatesPage{}, nil
	}

	var items []twins.State
	for k, v := range srm.states {
		if (uint64)(len(items)) >= limit {
			break
		}
		if !strings.HasPrefix(k, twinID) {
			continue
		}
		id := uint64(v.ID)
		if id >= offset && id < offset+limit {
			items = append(items, v)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	page := twins.StatesPage{
		States: items,
		PageMetadata: twins.PageMetadata{
			Total:  srm.total(twinID),
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (srm *stateRepositoryMock) total(twinID string) uint64 {
	var total uint64
	for k := range srm.states {
		if strings.HasPrefix(k, twinID) {
			total++
		}
	}
	return total
}

// RetrieveLast returns the last state related to twin spec by id.
func (srm *stateRepositoryMock) RetrieveLast(ctx context.Context, twinID string) (twins.State, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	items := make([]twins.State, 0)
	for _, v := range srm.states {
		if v.TwinID == twinID {
			items = append(items, v)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	if len(items) > 0 {
		return items[len(items)-1], nil
	}
	return twins.State{}, nil
}
