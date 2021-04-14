// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"errors"
	"sync"

	"github.com/mainflux/mainflux/lora"
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

func (trm *routeMapMock) Save(mfxID, extID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	trm.routes[extID] = mfxID
	trm.routes[mfxID] = extID
	return nil
}

func (trm *routeMapMock) Get(extID string) (string, error) {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	id, ok := trm.routes[extID]
	if !ok {
		return "", errors.New("route-map not found")
	}

	return id, nil
}

func (trm *routeMapMock) Remove(extID string) error {
	trm.mu.Lock()
	defer trm.mu.Unlock()

	var mfxID string
	for i, val := range trm.routes {
		if val == extID {
			mfxID = val
			delete(trm.routes, i)
		}
	}

	for i, val := range trm.routes {
		if val == mfxID {
			delete(trm.routes, i)
			return nil
		}
	}

	return nil
}
