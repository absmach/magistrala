// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/authn"
)

var _ authn.KeyRepository = (*keyRepositoryMock)(nil)

type keyRepositoryMock struct {
	mu   sync.Mutex
	keys map[string]authn.Key
}

// NewKeyRepository creates in-memory user repository
func NewKeyRepository() authn.KeyRepository {
	return &keyRepositoryMock{
		keys: make(map[string]authn.Key),
	}
}

func (krm *keyRepositoryMock) Save(ctx context.Context, key authn.Key) (string, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if _, ok := krm.keys[key.ID]; ok {
		return "", authn.ErrConflict
	}

	krm.keys[key.ID] = key
	return key.ID, nil
}
func (krm *keyRepositoryMock) Retrieve(ctx context.Context, issuerID, id string) (authn.Key, error) {
	krm.mu.Lock()
	defer krm.mu.Unlock()

	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		return key, nil
	}

	return authn.Key{}, authn.ErrNotFound
}
func (krm *keyRepositoryMock) Remove(ctx context.Context, issuerID, id string) error {
	krm.mu.Lock()
	defer krm.mu.Unlock()
	if key, ok := krm.keys[id]; ok && key.IssuerID == issuerID {
		delete(krm.keys, id)
	}
	return nil
}
