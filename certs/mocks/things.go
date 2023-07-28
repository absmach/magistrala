// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strconv"
	"sync"

	mfclients "github.com/mainflux/mainflux/pkg/clients"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/clients"
	upolicies "github.com/mainflux/mainflux/users/policies"
)

var _ clients.Service = (*mainfluxThings)(nil)

type mainfluxThings struct {
	mu      sync.Mutex
	counter uint64
	things  map[string]mfclients.Client
	auth    upolicies.AuthServiceClient
}

// NewThingsService returns Mainflux Things service mock.
// Only methods used by SDK are mocked.
func NewThingsService(things map[string]mfclients.Client, auth upolicies.AuthServiceClient) clients.Service {
	return &mainfluxThings{
		things: things,
		auth:   auth,
	}
}

func (svc *mainfluxThings) CreateThings(ctx context.Context, token string, ths ...mfclients.Client) ([]mfclients.Client, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token})
	if err != nil {
		return []mfclients.Client{}, errors.ErrAuthentication
	}
	for i := range ths {
		svc.counter++
		ths[i].Owner = userID.GetId()
		ths[i].ID = strconv.FormatUint(svc.counter, 10)
		ths[i].Credentials.Secret = ths[i].ID
		svc.things[ths[i].ID] = ths[i]
	}

	return ths, nil
}

func (svc *mainfluxThings) ViewClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token})
	if err != nil {
		return mfclients.Client{}, errors.ErrAuthentication
	}

	if t, ok := svc.things[id]; ok && t.Owner == userID.GetId() {
		return t, nil

	}

	return mfclients.Client{}, errors.ErrNotFound
}

func (svc *mainfluxThings) EnableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token})
	if err != nil {
		return mfclients.Client{}, errors.ErrAuthentication
	}

	if t, ok := svc.things[id]; !ok || t.Owner != userID.GetId() {
		return mfclients.Client{}, errors.ErrNotFound
	}
	if t, ok := svc.things[id]; ok && t.Owner == userID.GetId() {
		t.Status = mfclients.EnabledStatus
		return t, nil
	}
	return mfclients.Client{}, nil
}

func (svc *mainfluxThings) DisableClient(ctx context.Context, token, id string) (mfclients.Client, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()

	userID, err := svc.auth.Identify(ctx, &upolicies.IdentifyReq{Token: token})
	if err != nil {
		return mfclients.Client{}, errors.ErrAuthentication
	}

	if t, ok := svc.things[id]; !ok || t.Owner != userID.GetId() {
		return mfclients.Client{}, errors.ErrNotFound
	}
	if t, ok := svc.things[id]; ok && t.Owner == userID.GetId() {
		t.Status = mfclients.DisabledStatus
		return t, nil
	}
	return mfclients.Client{}, nil
}

func (svc *mainfluxThings) UpdateClient(context.Context, string, mfclients.Client) (mfclients.Client, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateClientSecret(context.Context, string, string, string) (mfclients.Client, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateClientOwner(context.Context, string, mfclients.Client) (mfclients.Client, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) UpdateClientTags(context.Context, string, mfclients.Client) (mfclients.Client, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListClients(context.Context, string, mfclients.Page) (mfclients.ClientsPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ListClientsByGroup(context.Context, string, string, mfclients.Page) (mfclients.MembersPage, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) Identify(context.Context, string) (string, error) {
	panic("not implemented")
}

func (svc *mainfluxThings) ShareClient(ctx context.Context, token, userID, groupID, thingID string, actions []string) error {
	panic("not implemented")
}
