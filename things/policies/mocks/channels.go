// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"strings"
	"sync"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/things/policies"
)

const separator = ":"

type cacheMock struct {
	mu       sync.Mutex
	policies map[string]string
}

// NewCache returns mock cache instance.
func NewCache() policies.Cache {
	return &cacheMock{
		policies: make(map[string]string),
	}
}

func (ccm *cacheMock) Put(_ context.Context, policy policies.CachedPolicy) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	key, value := kv(policy)
	ccm.policies[key] = value

	return nil
}

func (ccm *cacheMock) Get(_ context.Context, policy policies.CachedPolicy) (policies.CachedPolicy, error) {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	key, _ := kv(policy)

	val := ccm.policies[key]
	if val == "" {
		return policies.CachedPolicy{}, errors.ErrNotFound
	}

	thingID := extractThingID(val)
	if thingID == "" {
		return policies.CachedPolicy{}, errors.ErrNotFound
	}

	policy.Actions = separateActions(val)
	policy.ThingID = thingID

	return policy, nil
}

func (ccm *cacheMock) Remove(_ context.Context, policy policies.CachedPolicy) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	key, _ := kv(policy)

	delete(ccm.policies, key)

	return nil
}

// kv is used to create a key-value pair for caching.
func kv(p policies.CachedPolicy) (string, string) {
	key := p.ThingKey + separator + p.ChannelID
	val := strings.Join(p.Actions, separator)

	if p.ThingID != "" {
		val += separator + p.ThingID
	}

	return key, val
}

// separateActions is used to separate the actions from the cache values.
func separateActions(actions string) []string {
	return strings.Split(actions, separator)
}

// extractThingID is used to extract the thingID from the cache values.
func extractThingID(actions string) string {
	var lastIdx = strings.LastIndex(actions, separator)

	thingID := actions[lastIdx+1:]
	// check if the thingID is a valid UUID
	if len(thingID) != 36 {
		return ""
	}

	return thingID
}
