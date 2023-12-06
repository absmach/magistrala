// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"encoding/json"
	"sync"

	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
)

var _ readers.MessageRepository = (*messageRepositoryMock)(nil)

type messageRepositoryMock struct {
	mutex    sync.Mutex
	messages map[string][]readers.Message
}

// NewMessageRepository returns mock implementation of message repository.
func NewMessageRepository(chanID string, messages []readers.Message) readers.MessageRepository {
	repo := map[string][]readers.Message{
		chanID: messages,
	}

	return &messageRepositoryMock{
		mutex:    sync.Mutex{},
		messages: repo,
	}
}

func (repo *messageRepositoryMock) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	if rpm.Format != "" && rpm.Format != "messages" {
		return readers.MessagesPage{}, nil
	}

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return readers.MessagesPage{}, err
	}
	if err := json.Unmarshal(meta, &query); err != nil {
		return readers.MessagesPage{}, err
	}

	var msgs []readers.Message
	for _, m := range repo.messages[chanID] {
		msg := m.(senml.Message)

		ok := true

		for name := range query {
			switch name {
			case "subtopic":
				if rpm.Subtopic != msg.Subtopic {
					ok = false
				}
			case "publisher":
				if rpm.Publisher != msg.Publisher {
					ok = false
				}
			case "name":
				if rpm.Name != msg.Name {
					ok = false
				}
			case "protocol":
				if rpm.Protocol != msg.Protocol {
					ok = false
				}
			case "v":
				if msg.Value == nil {
					ok = false
				}

				val, okQuery := query["comparator"]
				if okQuery {
					switch val.(string) {
					case readers.LowerThanKey:
						if msg.Value != nil &&
							*msg.Value >= rpm.Value {
							ok = false
						}
					case readers.LowerThanEqualKey:
						if msg.Value != nil &&
							*msg.Value > rpm.Value {
							ok = false
						}
					case readers.GreaterThanKey:
						if msg.Value != nil &&
							*msg.Value <= rpm.Value {
							ok = false
						}
					case readers.GreaterThanEqualKey:
						if msg.Value != nil &&
							*msg.Value < rpm.Value {
							ok = false
						}
					case readers.EqualKey:
					default:
						if msg.Value != nil &&
							*msg.Value != rpm.Value {
							ok = false
						}
					}
				}
			case "vb":
				if msg.BoolValue == nil ||
					(msg.BoolValue != nil &&
						*msg.BoolValue != rpm.BoolValue) {
					ok = false
				}
			case "vs":
				if msg.StringValue == nil ||
					(msg.StringValue != nil &&
						*msg.StringValue != rpm.StringValue) {
					ok = false
				}
			case "vd":
				if msg.DataValue == nil ||
					(msg.DataValue != nil &&
						*msg.DataValue != rpm.DataValue) {
					ok = false
				}
			case "from":
				if msg.Time < rpm.From {
					ok = false
				}
			case "to":
				if msg.Time >= rpm.To {
					ok = false
				}
			}

			if !ok {
				break
			}
		}

		if ok {
			msgs = append(msgs, m)
		}
	}

	numOfMessages := uint64(len(msgs))

	if rpm.Offset >= numOfMessages {
		return readers.MessagesPage{}, nil
	}

	if rpm.Limit < 1 {
		return readers.MessagesPage{}, nil
	}

	end := rpm.Offset + rpm.Limit
	if rpm.Offset+rpm.Limit > numOfMessages {
		end = numOfMessages
	}

	return readers.MessagesPage{
		PageMetadata: rpm,
		Total:        uint64(len(msgs)),
		Messages:     msgs[rpm.Offset:end],
	}, nil
}
