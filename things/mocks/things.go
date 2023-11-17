// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/things"
)

type clientCacheMock struct {
	mu     sync.Mutex
	things map[string]string
}

// NewCache returns mock cache instance.
func NewCache() things.Cache {
	return &clientCacheMock{
		things: make(map[string]string),
	}
}

func (tcm *clientCacheMock) Save(_ context.Context, key, id string) error {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	tcm.things[key] = id
	return nil
}

func (tcm *clientCacheMock) ID(_ context.Context, key string) (string, error) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	id, ok := tcm.things[key]
	if !ok {
		return "", errors.ErrNotFound
	}

	return id, nil
}

func (tcm *clientCacheMock) Remove(_ context.Context, id string) error {
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
