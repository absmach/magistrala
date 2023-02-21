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
	certsByID      map[string]certs.Cert
	certsByThingID map[string]map[string][]certs.Cert
}

// NewCertsRepository creates in-memory certs repository.
func NewCertsRepository() certs.Repository {
	return &certsRepoMock{
		certsByID:      make(map[string]certs.Cert),
		certsByThingID: make(map[string]map[string][]certs.Cert),
	}
}

func (c *certsRepoMock) Save(ctx context.Context, cert certs.Cert) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	crt := cert

	_, ok := c.certsByThingID[cert.OwnerID][cert.ThingID]
	switch ok {
	case false:
		c.certsByThingID[cert.OwnerID] = map[string][]certs.Cert{
			cert.ThingID: []certs.Cert{crt},
		}
	default:
		c.certsByThingID[cert.OwnerID][cert.ThingID] = append(c.certsByThingID[cert.OwnerID][cert.ThingID], crt)
	}

	c.certsByID[cert.ID] = crt
	c.counter++
	return nil
}

func (c *certsRepoMock) Retrieve(ctx context.Context, ownerID, certID, name, thingID, serial string, status certs.Status, offset uint64, limit int64) (certs.Page, error) {
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

			switch limit >= 0 {
			case true:
				if uint64(i) >= offset && uint64(i) < offset+uint64(limit) {
					crts = append(crts, v)
				}
			default:
				crts = append(crts, v)

			}

		}
	}

	total, err := c.RetrieveCount(ctx, ownerID, certID, name, thingID, serial, status)
	if err != nil {
		return certs.Page{}, err
	}

	page := certs.Page{
		Certs:  crts,
		Total:  total,
		Offset: offset,
		Limit:  limit,
	}
	return page, nil
}

func (c *certsRepoMock) RetrieveCount(ctx context.Context, ownerID, certID, name, thingID, serial string, status certs.Status) (uint64, error) {
	return c.counter, nil
}

func (c *certsRepoMock) Remove(ctx context.Context, ownerID, certID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	crt, ok := c.certsByID[certID]
	if !ok {
		return errors.ErrNotFound
	}
	delete(c.certsByID, crt.ID)
	delete(c.certsByThingID, crt.ThingID)
	return nil
}

func (c *certsRepoMock) Update(ctx context.Context, oldSerial string, cert certs.Cert) error {
	return nil
}

func (c *certsRepoMock) RetrieveThingCerts(ctx context.Context, thingID string) (certs.Page, error) {
	return certs.Page{}, nil
}

func (c *certsRepoMock) RemoveThingCerts(ctx context.Context, thingID string) error {
	return nil
}
