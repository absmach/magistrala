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
	"github.com/mainflux/mainflux/readers"
)

var _ readers.MessageRepository = (*messageRepositoryMock)(nil)

type messageRepositoryMock struct {
	mutex    sync.Mutex
	messages map[string][]mainflux.Message
}

// NewMessageRepository returns mock implementation of message repository.
func NewMessageRepository(messages map[string][]mainflux.Message) readers.MessageRepository {
	return &messageRepositoryMock{
		mutex:    sync.Mutex{},
		messages: messages,
	}
}

func (repo *messageRepositoryMock) ReadAll(chanID string, offset, limit uint64) []mainflux.Message {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	end := offset + limit

	numOfMessages := uint64(len(repo.messages[chanID]))
	if offset < 0 || offset >= numOfMessages {
		return []mainflux.Message{}
	}

	if limit < 1 {
		return []mainflux.Message{}
	}

	if offset+limit > numOfMessages {
		end = numOfMessages
	}

	return repo.messages[chanID][offset:end]
}
