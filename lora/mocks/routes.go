// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"errors"
	"sync"

	"github.com/absmach/magistrala/lora"
)

type routeMapMock struct {
	mu     sync.Mutex
	routes map[string]string
}

// NewRouteMap returns mock route-map instance.
func NewRouteMap() lora.RouteMapRepository {
	return &routeMapMock{
		routes: make(map[string]string),
	}
}

func (trm *routeMapMock) Save(_ context.Context, mgxID, extID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.routes[extID] = mgxID
	trm.routes[mgxID] = extID
	return nil
}

func (trm *routeMapMock) Get(_ context.Context, extID string) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	id, ok := trm.routes[extID]
	if !ok {
		return "", errors.New("route-map not found")
	}

	return id, nil
}

func (trm *routeMapMock) Remove(_ context.Context, extID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	var mgxID string
	for i, val := range trm.routes {
		if val == extID {
			mgxID = val
			delete(trm.routes, i)
		}
	}

	for i, val := range trm.routes {
		if val == mgxID {
			delete(trm.routes, i)
			return nil
		}
	}

	return nil
}
