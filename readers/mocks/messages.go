// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"encoding/json"
	"sync"

	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
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
	meta, _ := json.Marshal(rpm)
	json.Unmarshal(meta, &query)

	var msgs []readers.Message
	for _, m := range repo.messages[chanID] {
		senml := m.(senml.Message)

		ok := true

		for name := range query {
			switch name {
			case "subtopic":
				if rpm.Subtopic != senml.Subtopic {
					ok = false
				}
			case "publisher":
				if rpm.Publisher != senml.Publisher {
					ok = false
				}
			case "name":
				if rpm.Name != senml.Name {
					ok = false
				}
			case "protocol":
				if rpm.Protocol != senml.Protocol {
					ok = false
				}
			case "v":
				if senml.Value == nil {
					ok = false
				}

				val, okQuery := query["comparator"]
				if okQuery {
					switch val.(string) {
					case readers.LowerThanKey:
						if senml.Value != nil &&
							*senml.Value >= rpm.Value {
							ok = false
						}
					case readers.LowerThanEqualKey:
						if senml.Value != nil &&
							*senml.Value > rpm.Value {
							ok = false
						}
					case readers.GreaterThanKey:
						if senml.Value != nil &&
							*senml.Value <= rpm.Value {
							ok = false
						}
					case readers.GreaterThanEqualKey:
						if senml.Value != nil &&
							*senml.Value < rpm.Value {
							ok = false
						}
					case readers.EqualKey:
					default:
						if senml.Value != nil &&
							*senml.Value != rpm.Value {
							ok = false
						}
					}
				}
			case "vb":
				if senml.BoolValue == nil ||
					(senml.BoolValue != nil &&
						*senml.BoolValue != rpm.BoolValue) {
					ok = false
				}
			case "vs":
				if senml.StringValue == nil ||
					(senml.StringValue != nil &&
						*senml.StringValue != rpm.StringValue) {
					ok = false
				}
			case "vd":
				if senml.DataValue == nil ||
					(senml.DataValue != nil &&
						*senml.DataValue != rpm.DataValue) {
					ok = false
				}
			case "from":
				if senml.Time < rpm.From {
					ok = false
				}
			case "to":
				if senml.Time >= rpm.To {
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
