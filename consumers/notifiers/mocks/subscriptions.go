// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"sort"
	"sync"

	"github.com/absmach/magistrala/consumers/notifiers"
	"github.com/absmach/magistrala/pkg/errors"
)

var _ notifiers.SubscriptionsRepository = (*subRepoMock)(nil)

type subRepoMock struct {
	mu   sync.Mutex
	subs map[string]notifiers.Subscription
}

// NewRepo returns a new Subscriptions repository mock.
func NewRepo(subs map[string]notifiers.Subscription) notifiers.SubscriptionsRepository {
	return &subRepoMock{
		subs: subs,
	}
}

func (srm *subRepoMock) Save(_ context.Context, sub notifiers.Subscription) (string, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()
	if _, ok := srm.subs[sub.ID]; ok {
		return "", errors.ErrConflict
	}
	for _, s := range srm.subs {
		if s.Contact == sub.Contact && s.Topic == sub.Topic {
			return "", errors.ErrConflict
		}
	}

	srm.subs[sub.ID] = sub
	return sub.ID, nil
}

func (srm *subRepoMock) Retrieve(_ context.Context, id string) (notifiers.Subscription, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()
	ret, ok := srm.subs[id]
	if !ok {
		return notifiers.Subscription{}, errors.ErrNotFound
	}
	return ret, nil
}

func (srm *subRepoMock) RetrieveAll(_ context.Context, pm notifiers.PageMetadata) (notifiers.Page, error) {
	srm.mu.Lock()
	defer srm.mu.Unlock()

	// Sort keys
	keys := make([]string, 0)
	for k := range srm.subs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var subs []notifiers.Subscription
	var total int
	offset := int(pm.Offset)
	for _, k := range keys {
		v := srm.subs[k]
		if pm.Topic == "" {
			if pm.Contact == "" {
				if total < offset {
					total++
					continue
				}
				total++
				subs = appendSubs(subs, v, pm.Limit)
				continue
			}
			if pm.Contact == v.Contact {
				if total < offset {
					total++
					continue
				}
				total++
				subs = appendSubs(subs, v, pm.Limit)
				continue
			}
		}
		if pm.Topic == v.Topic {
			if pm.Contact == "" || pm.Contact == v.Contact {
				if total < offset {
					total++
					continue
				}
				total++
				subs = appendSubs(subs, v, pm.Limit)
			}
		}
	}

	if len(subs) == 0 {
		return notifiers.Page{}, errors.ErrNotFound
	}

	ret := notifiers.Page{
		PageMetadata:  pm,
		Total:         uint(total),
		Subscriptions: subs,
	}

	return ret, nil
}

func appendSubs(subs []notifiers.Subscription, sub notifiers.Subscription, max int) []notifiers.Subscription {
	if len(subs) < max || max == -1 {
		subs = append(subs, sub)
	}
	return subs
}

func (srm *subRepoMock) Remove(_ context.Context, id string) error {
	srm.mu.Lock()
	defer srm.mu.Unlock()
	if _, ok := srm.subs[id]; !ok {
		return errors.ErrNotFound
	}
	delete(srm.subs, id)
	return nil
}
