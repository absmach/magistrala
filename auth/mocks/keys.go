// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
)

var _ auth.KeyRepository = (*keyRepositoryMock)(nil)

type keyRepositoryMock struct {
	mu   sync.Mutex
	keys map[string]auth.Key
}

// NewKeyRepository creates in-memory user repository
func NewKeyRepository() auth.KeyRepository {
	return &keyRepositoryMock{
		keys: make(map[string]auth.Key),
	}
}

func (krm *keyRepositoryMock) Save(ctx context.Context, key auth.Key) (string, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if _, ok := krm.keys[key.ID]; ok {
		return "", errors.ErrConflict
	}

	krm.keys[key.ID] = key
	return key.ID, nil
}
func (krm *keyRepositoryMock) RetrieveByID(ctx context.Context, issuerID, id string) (auth.Key, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		return key, nil
	}

	return auth.Key{}, errors.ErrNotFound
}

func (krm *keyRepositoryMock) RetrieveAll(ctx context.Context, issuerID string, pm auth.PageMetadata) (auth.KeyPage, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	kp := auth.KeyPage{}
	i := uint64(0)

	for _, k := range krm.keys {
		if i >= pm.Offset && i < (pm.Limit+pm.Offset) {
			kp.Keys = append(kp.Keys, k)
		}
		i++
	}

	kp.Offset = pm.Offset
	kp.Limit = pm.Limit
	kp.Total = uint64(i)

	return kp, nil
}

func (krm *keyRepositoryMock) Remove(ctx context.Context, issuerID, id string) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		delete(krm.keys, id)
	}
	return nil
}
