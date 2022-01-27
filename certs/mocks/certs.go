// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sync"

	"github.com/mainflux/mainflux/certs"
	"github.com/mainflux/mainflux/pkg/errors"
)

var _ certs.Repository = (*certsRepoMock)(nil)

type certsRepoMock struct {
	mu             sync.Mutex
	counter        uint64
	certsBySerial  map[string]certs.Cert
	certsByThingID map[string]map[string][]certs.Cert
}

// NewCertsRepository creates in-memory certs repository.
func NewCertsRepository() certs.Repository {
	return &certsRepoMock{
		certsBySerial:  make(map[string]certs.Cert),
		certsByThingID: make(map[string]map[string][]certs.Cert),
	}
}

func (c *certsRepoMock) Save(ctx context.Context, cert certs.Cert) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt := certs.Cert{
		OwnerID: cert.OwnerID,
		ThingID: cert.ThingID,
		Serial:  cert.Serial,
		Expire:  cert.Expire,
	}

	_, ok := c.certsByThingID[cert.OwnerID][cert.ThingID]
	switch ok {
	case false:
		c.certsByThingID[cert.OwnerID] = map[string][]certs.Cert{
			cert.ThingID: []certs.Cert{crt},
		}
	default:
		c.certsByThingID[cert.OwnerID][cert.ThingID] = append(c.certsByThingID[cert.OwnerID][cert.ThingID], crt)
	}

	c.certsBySerial[cert.Serial] = crt
	c.counter++
	return cert.Serial, nil
}

func (c *certsRepoMock) RetrieveAll(ctx context.Context, ownerID string, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if limit <= 0 {
		return certs.Page{}, nil
	}

	oc, ok := c.certsByThingID[ownerID]
	if !ok {
		return certs.Page{}, errors.ErrNotFound
	}

	var crts []certs.Cert
	for _, tc := range oc {
		for i, v := range tc {
			if uint64(i) >= offset && uint64(i) < offset+limit {
				crts = append(crts, v)
			}
		}
	}

	page := certs.Page{
		Certs:  crts,
		Total:  c.counter,
		Offset: offset,
		Limit:  limit,
	}
	return page, nil
}

func (c *certsRepoMock) Remove(ctx context.Context, ownerID, serial string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	crt, ok := c.certsBySerial[serial]
	if !ok {
		return errors.ErrNotFound
	}
	delete(c.certsBySerial, crt.Serial)
	delete(c.certsByThingID, crt.ThingID)
	return nil
}

func (c *certsRepoMock) RetrieveByThing(ctx context.Context, ownerID, thingID string, offset, limit uint64) (certs.Page, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if limit <= 0 {
		return certs.Page{}, nil
	}

	cs, ok := c.certsByThingID[ownerID][thingID]
	if !ok {
		return certs.Page{}, errors.ErrNotFound
	}

	var crts []certs.Cert
	for i, v := range cs {
		if uint64(i) >= offset && uint64(i) < offset+limit {
			crts = append(crts, v)
		}
	}

	page := certs.Page{
		Certs:  crts,
		Total:  c.counter,
		Offset: offset,
		Limit:  limit,
	}
	return page, nil
}

func (c *certsRepoMock) RetrieveBySerial(ctx context.Context, ownerID, serialID string) (certs.Cert, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt, ok := c.certsBySerial[serialID]
	if !ok {
		return certs.Cert{}, errors.ErrNotFound
	}

	return crt, nil
}
