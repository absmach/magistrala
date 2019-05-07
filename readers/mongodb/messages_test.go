//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mainflux/mainflux/readers"
	mreaders "github.com/mainflux/mainflux/readers/mongodb"
	mwriters "github.com/mainflux/mainflux/writers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux"

	log "github.com/mainflux/mainflux/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDB      = "test"
	collection  = "mainflux"
	chanID      = "1"
	subtopic    = "subtopic"
	msgsNum     = 42
	valueFields = 6
)

var (
	port string
	addr string
	msg  = mainflux.Message{
		Channel:   chanID,
		Publisher: "1",
		Protocol:  "mqtt",
	}
	testLog, _ = log.New(os.Stdout, log.Info.String())
)

func TestReadAll(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	writer := mwriters.New(db)

	messages := []mainflux.Message{}
	subtopicMsgs := []mainflux.Message{}
	now := time.Now().Unix()
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		msg.Subtopic = ""
		switch count {
		case 0:
			msg.Subtopic = subtopic
			msg.Value = &mainflux.Message_FloatValue{FloatValue: 5}
		case 1:
			msg.Value = &mainflux.Message_BoolValue{BoolValue: false}
		case 2:
			msg.Value = &mainflux.Message_StringValue{StringValue: "value"}
		case 3:
			msg.Value = &mainflux.Message_DataValue{DataValue: "base64data"}
		case 4:
			msg.ValueSum = nil
		case 5:
			msg.ValueSum = &mainflux.SumValue{Value: 45}
		}
		msg.Time = float64(now - int64(i))

		err := writer.Save(msg)
		require.Nil(t, err, fmt.Sprintf("failed to store message to MongoDB: %s", err))
		messages = append(messages, msg)
		if count == 0 {
			subtopicMsgs = append(subtopicMsgs, msg)
		}
	}

	reader := mreaders.New(db)

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
			limit:  10,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    10,
				Messages: messages[0:10],
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
				Messages: []mainflux.Message{},
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
				Messages: messages[40:42],
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
				Messages: []mainflux.Message{},
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
				Messages: subtopicMsgs,
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
