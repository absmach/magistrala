// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	mwriter "github.com/absmach/magistrala/consumers/writers/mongodb"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	mreader "github.com/absmach/magistrala/readers/mongodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testDB      = "test"
	subtopic    = "subtopic"
	msgsNum     = 100
	limit       = 10
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
	wrongID     = "wrong-id"

	format1 = "format_1"
	format2 = "format_2"
)

var (
	port string
	addr string

	v   float64 = 5
	vs          = "stringValue"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42
)

func TestReadSenml(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	writer := mwriter.New(db)

	chanID := testsutil.GenerateUUID(t)
	pubID := testsutil.GenerateUUID(t)
	pubID2 := testsutil.GenerateUUID(t)
	wrongID := testsutil.GenerateUUID(t)

	m := senml.Message{
		Channel:   chanID,
		Publisher: pubID,
		Protocol:  mqttProt,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := time.Now().Unix()

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = float64(now - int64(i))

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
			valueMsgs = append(valueMsgs, msg)
		case 1:
			msg.BoolValue = &vb
			boolMsgs = append(boolMsgs, msg)
		case 2:
			msg.StringValue = &vs
			stringMsgs = append(stringMsgs, msg)
		case 3:
			msg.DataValue = &vd
			dataMsgs = append(dataMsgs, msg)
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Publisher = pubID2
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}
		messages = append(messages, msg)
	}
	err = writer.ConsumeBlocking(context.TODO(), messages)
	require.Nil(t, err, fmt.Sprintf("failed to store message to MongoDB: %s", err))
	reader := mreader.New(db)

	cases := map[string]struct {
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages),
			},
		},
		"read message page for non-existent channel": {
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromSenml(messages[msgsNum-20 : msgsNum]),
			},
		},
		"read message with non-existent subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    msgsNum,
				Subtopic: "not-present",
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read message with subtopic": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Subtopic: subtopic,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with publisher": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with invalid format": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Format: "messagess",
				Offset: 0,
				Limit:  uint64(len(queryMsgs)),
			},
			page: readers.MessagesPage{
				Total:    0,
				Messages: []readers.Message{},
			},
		},
		"read message with protocol": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:   0,
				Limit:    uint64(len(queryMsgs)),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs),
			},
		},
		"read message with name": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				Name:   msgName,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(queryMsgs[0:limit]),
			},
		},
		"read message with value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				Value:  v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(queryMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with value and equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     0,
				Limit:      limit,
				Value:      v,
				Comparator: readers.EqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with value and lower-than comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     0,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with value and lower-than-or-equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     0,
				Limit:      limit,
				Value:      v + 1,
				Comparator: readers.LowerThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with value and greater-than comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     0,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with value and greater-than-or-equal comparator": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:     0,
				Limit:      limit,
				Value:      v - 1,
				Comparator: readers.GreaterThanEqualKey,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		"read message with boolean value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     limit,
				BoolValue: vb,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(boolMsgs)),
				Messages: fromSenml(boolMsgs[0:limit]),
			},
		},
		"read message with string value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:      0,
				Limit:       limit,
				StringValue: vs,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(stringMsgs)),
				Messages: fromSenml(stringMsgs[0:limit]),
			},
		},
		"read message with data value": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset:    0,
				Limit:     limit,
				DataValue: vd,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(dataMsgs)),
				Messages: fromSenml(dataMsgs[0:limit]),
			},
		},
		"read message with from": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[0:21])),
				From:   messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0:21])),
				Messages: fromSenml(messages[0:21]),
			},
		},
		"read message with to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[21:])),
				To:     messages[20].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[21:])),
				Messages: fromSenml(messages[21:]),
			},
		},
		"read message with from/to": {
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				From:   messages[5].Time,
				To:     messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    5,
				Messages: fromSenml(messages[1:6]),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: got incorrect list of senml Messages from ReadAll()", desc))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %v got %v", desc, tc.page.Total, result.Total))
	}
}

func TestReadJSON(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	writer := mwriter.New(db)

	id1 := testsutil.GenerateUUID(t)
	m := json.Message{
		Channel:   id1,
		Publisher: id1,
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "coap",
		Payload: map[string]interface{}{
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42.0,
			},
		},
	}
	messages1 := json.Messages{
		Format: format1,
	}
	msgs1 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		msg := m
		messages1.Data = append(messages1.Data, msg)
		m := toMap(msg)
		msgs1 = append(msgs1, m)
	}

	err = writer.ConsumeBlocking(context.TODO(), messages1)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	id2 := testsutil.GenerateUUID(t)
	m = json.Message{
		Channel:   id2,
		Publisher: id2,
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/other_format/some_other_json",
		Protocol:  "udp",
		Payload: map[string]interface{}{
			"field_2": "other_value",
			"field_3": false,
			"field_5": map[string]interface{}{
				"field_1": "wrong_value",
				"field_2": 24.5,
			},
		},
	}
	messages2 := json.Messages{
		Format: format2,
	}
	msgs2 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		msg := m
		if i%2 == 0 {
			msg.Protocol = httpProt
		}
		messages2.Data = append(messages2.Data, msg)
		m := toMap(msg)
		msgs2 = append(msgs2, m)
	}

	err = writer.ConsumeBlocking(context.TODO(), messages2)
	require.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))

	httpMsgs := []map[string]interface{}{}
	for i := 0; i < msgsNum; i += 2 {
		httpMsgs = append(httpMsgs, msgs2[i])
	}
	reader := mreader.New(db)

	cases := map[string]struct {
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		"read message page for existing channel": {
			chanID: id1,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: 0,
				Limit:  10,
			},
			page: readers.MessagesPage{
				Total:    100,
				Messages: fromJSON(msgs1[:10]),
			},
		},
		"read message page for non-existent channel": {
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: 0,
				Limit:  10,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		"read message last page": {
			chanID: id2,
			pageMeta: readers.PageMetadata{
				Format: messages2.Format,
				Offset: msgsNum - 20,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs2[msgsNum-20 : msgsNum]),
			},
		},
		"read message with protocol": {
			chanID: id2,
			pageMeta: readers.PageMetadata{
				Format:   messages2.Format,
				Offset:   0,
				Limit:    uint64(msgsNum / 2),
				Protocol: httpProt,
			},
			page: readers.MessagesPage{
				Total:    uint64(msgsNum / 2),
				Messages: fromJSON(httpMsgs),
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.pageMeta)

		for i := 0; i < len(result.Messages); i++ {
			m := result.Messages[i]
			// Remove id as it is not sent by the client.
			delete(m.(map[string]interface{}), "_id")
			result.Messages[i] = m
		}
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: got incorrect list of json Messages from ReadAll()", desc))
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

func fromJSON(msg []map[string]interface{}) []readers.Message {
	var ret []readers.Message
	for _, m := range msg {
		ret = append(ret, m)
	}
	return ret
}

func toMap(msg json.Message) map[string]interface{} {
	return map[string]interface{}{
		"channel":   msg.Channel,
		"created":   msg.Created,
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}(msg.Payload),
	}
}
