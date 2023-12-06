// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/twins"
)

var _ twins.TwinRepository = (*twinRepositoryMock)(nil)

type twinRepositoryMock struct {
	mu    sync.Mutex
	twins map[string]twins.Twin
}

// NewTwinRepository creates in-memory twin repository.
func NewTwinRepository() twins.TwinRepository {
	return &twinRepositoryMock{
		twins: make(map[string]twins.Twin),
	}
}

func (trm *twinRepositoryMock) Save(ctx context.Context, twin twins.Twin) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for _, tw := range trm.twins {
		if tw.ID == twin.ID {
			return "", errors.ErrConflict
		}
	}

	trm.twins[key(twin.Owner, twin.ID)] = twin

	return twin.ID, nil
}

func (trm *twinRepositoryMock) Update(ctx context.Context, twin twins.Twin) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	dbKey := key(twin.Owner, twin.ID)
	if _, ok := trm.twins[dbKey]; !ok {
		return errors.ErrNotFound
	}

	trm.twins[dbKey] = twin

	return nil
}

func (trm *twinRepositoryMock) RetrieveByID(_ context.Context, twinID string) (twins.Twin, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for k, v := range trm.twins {
		if twinID == v.ID {
			return trm.twins[k], nil
		}
	}

	return twins.Twin{}, errors.ErrNotFound
}

func (trm *twinRepositoryMock) RetrieveByAttribute(ctx context.Context, channel, subtopic string) ([]string, error) {
	var ids []string
	for _, twin := range trm.twins {
		def := twin.Definitions[len(twin.Definitions)-1]
		for _, attr := range def.Attributes {
			if attr.Channel == channel && (attr.Subtopic == twins.SubtopicWildcard || attr.Subtopic == subtopic) {
				ids = append(ids, twin.ID)
				break
			}
		}
	}
	return ids, nil
}

func (trm *twinRepositoryMock) RetrieveAll(_ context.Context, owner string, offset, limit uint64, name string, metadata twins.Metadata) (twins.Page, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	items := make([]twins.Twin, 0)

	if limit <= 0 {
		return twins.Page{}, nil
	}

	for k, v := range trm.twins {
		if (uint64)(len(items)) >= limit {
			break
		}
		if len(name) > 0 && v.Name != name {
			continue
		}
		if !strings.HasPrefix(k, owner) {
			continue
		}
		suffix := string(v.ID[len(uuid.Prefix):])
		id, _ := strconv.ParseUint(suffix, 10, 64)
		if id > offset && id <= offset+limit {
			items = append(items, v)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})

	total := uint64(len(trm.twins))
	page := twins.Page{
		Twins: items,
		PageMetadata: twins.PageMetadata{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return page, nil
}

func (trm *twinRepositoryMock) Remove(ctx context.Context, twinID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	for k, v := range trm.twins {
		if twinID == v.ID {
			delete(trm.twins, k)
			return nil
		}
	}

	return nil
}

type twinCacheMock struct {
	mu      sync.Mutex
	attrIds map[string]map[string]bool
	idAttrs map[string]map[string]bool
}

// NewTwinCache returns mock cache instance.
func NewTwinCache() twins.TwinCache {
	return &twinCacheMock{
		attrIds: make(map[string]map[string]bool),
		idAttrs: make(map[string]map[string]bool),
	}
}

func (tcm *twinCacheMock) Save(_ context.Context, twin twins.Twin) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	if len(twin.Definitions) < 1 {
		return nil
	}
	def := twin.Definitions[len(twin.Definitions)-1]
	tcm.save(def, twin.ID)

	return nil
}

func (tcm *twinCacheMock) SaveIDs(ctx context.Context, channel, subtopic string, ids []string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	for _, id := range ids {
		attrKey := channel + subtopic
		if _, ok := tcm.attrIds[attrKey]; !ok {
			tcm.attrIds[attrKey] = make(map[string]bool)
		}
		tcm.attrIds[attrKey][id] = true

		if _, ok := tcm.idAttrs[id]; !ok {
			tcm.idAttrs[id] = make(map[string]bool)
		}
		tcm.idAttrs[id][attrKey] = true
	}

	return nil
}

func (tcm *twinCacheMock) Update(_ context.Context, twin twins.Twin) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	if err := tcm.remove(twin.ID); err != nil {
		return nil
	}

	if len(twin.Definitions) < 1 {
		return nil
	}
	def := twin.Definitions[len(twin.Definitions)-1]
	tcm.save(def, twin.ID)
	return nil
}

func (tcm *twinCacheMock) IDs(_ context.Context, channel, subtopic string) ([]string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	var ids []string

	for k := range tcm.attrIds[channel+subtopic] {
		ids = append(ids, k)
	}
	for k := range tcm.attrIds[channel+twins.SubtopicWildcard] {
		ids = append(ids, k)
	}

	return ids, nil
}

func (tcm *twinCacheMock) Remove(_ context.Context, twinID string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	return tcm.remove(twinID)
}

func (tcm *twinCacheMock) remove(twinID string) error {
	attrKeys, ok := tcm.idAttrs[twinID]
	if !ok {
		return nil
	}

	delete(tcm.idAttrs, twinID)
	for attrKey := range attrKeys {
		delete(tcm.attrIds[attrKey], twinID)
	}
	return nil
}

func (tcm *twinCacheMock) save(def twins.Definition, twinID string) {
	for _, attr := range def.Attributes {
		attrKey := attr.Channel + attr.Subtopic
		if _, ok := tcm.attrIds[attrKey]; !ok {
			tcm.attrIds[attrKey] = make(map[string]bool)
		}
		tcm.attrIds[attrKey][twinID] = true

		idKey := twinID
		if _, ok := tcm.idAttrs[idKey]; !ok {
			tcm.idAttrs[idKey] = make(map[string]bool)
		}
		tcm.idAttrs[idKey][attrKey] = true
	}
}
