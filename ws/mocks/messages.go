//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mocks

import (
	"sync"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/ws"
)

var _ ws.Service = (*mockService)(nil)

type mockService struct {
	subscriptions map[uint64]*ws.Channel
	pubError      error
	mutex         sync.Mutex
}

// NewService returns mock message publisher.
func NewService(subs map[uint64]*ws.Channel, pubError error) ws.Service {
	return &mockService{subs, pubError, sync.Mutex{}}
}

func (svc *mockService) Publish(msg mainflux.RawMessage) error {
	if len(msg.Payload) == 0 {
		return svc.pubError
	}
	svc.mutex.Lock()
	defer svc.mutex.Unlock()
	svc.subscriptions[msg.Channel].Messages <- msg
	return nil
}

func (svc *mockService) Subscribe(chanID uint64, channel *ws.Channel) error {
	svc.mutex.Lock()
	defer svc.mutex.Unlock()
	if _, ok := svc.subscriptions[chanID]; !ok {
		return ws.ErrFailedSubscription
	}
	svc.subscriptions[chanID] = channel

	return nil
}
