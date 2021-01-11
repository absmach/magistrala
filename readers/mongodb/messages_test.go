// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	writer "github.com/mainflux/mainflux/consumers/writers/mongodb"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	log "github.com/mainflux/mainflux/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDB      = "test"
	collection  = "messages"
	chanID      = "1"
	subtopic    = "subtopic"
	msgsNum     = 42
	valueFields = 5
)

var (
	port string
	addr string
	m    = senml.Message{
		Channel:   chanID,
		Publisher: "1",
		Protocol:  "mqtt",
	}
	testLog, _ = log.New(os.Stdout, log.Info.String())
)
var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestReadSenml(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	writer := writer.New(db)

	messages := []senml.Message{}
	subtopicMsgs := []senml.Message{}
	now := time.Now().Unix()
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		msg := m
		switch count {
		case 0:
			msg.Subtopic = subtopic
			msg.Value = &v
		case 1:
			msg.BoolValue = &boolV
		case 2:
			msg.StringValue = &stringV
		case 3:
			msg.DataValue = &dataV
		case 4:
			msg.Sum = &sum
		}
		msg.Time = float64(now - int64(i))
		messages = append(messages, msg)
		if count == 0 {
			subtopicMsgs = append(subtopicMsgs, msg)
		}
	}
	err = writer.Consume(messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to MongoDB: %s", err))
	reader := reader.New(db)

	cases := map[string]struct {
		chanID string
		offset uint64
		limit  uint64
		query  map[string]string
		page   readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: chanID,
			offset: 0,
			limit:  11,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    11,
				Messages: fromSenml(messages[0:11]),
			},
		},
		"read message page for non-existent channel": {
			chanID: "2",
			offset: 0,
			limit:  10,
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			offset: 40,
			limit:  10,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   40,
				Limit:    10,
				Messages: fromSenml(messages[40:42]),
			},
		},
		"read message with non-existent subtopic": {
			chanID: chanID,
			offset: 0,
			limit:  msgsNum,
			query:  map[string]string{"subtopic": "not-present"},
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    msgsNum,
				Messages: []readers.Message{},
			},
		},
		"read message with subtopic": {
			chanID: chanID,
			offset: 0,
			limit:  10,
			query:  map[string]string{"subtopic": subtopic},
			page: readers.MessagesPage{
				Total:    uint64(len(subtopicMsgs)),
				Offset:   0,
				Limit:    10,
				Messages: fromSenml(subtopicMsgs),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.offset, tc.limit, tc.query)

		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
		ret = append(ret, m)
	}
	return ret
}
