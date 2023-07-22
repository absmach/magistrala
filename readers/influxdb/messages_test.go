// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package influxdb_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb-client-go/v2"
	iwriter "github.com/mainflux/mainflux/consumers/writers/influxdb"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/readers"
	ireader "github.com/mainflux/mainflux/readers/influxdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	subtopic    = "topic"
	msgsNum     = 100
	limit       = 10
	valueFields = 5
	mqttProt    = "mqtt"
	httpProt    = "http"
	msgName     = "temperature"
	offset      = 21
	format1     = "format1"
	format2     = "format2"
	wrongID     = "wrong_id"
)

var (
	v   float64 = 5
	vs  string  = "a"
	vb  bool    = true
	vd  string  = "dataValue"
	sum float64 = 42

	client  influxdata.Client
	repoCfg = struct {
		Bucket string
		Org    string
	}{
		Bucket: dbBucket,
		Org:    dbOrg,
	}
	idProvider = uuid.New()
)

func TestReadSenml(t *testing.T) {
	asyncWriter := iwriter.NewAsync(client, repoCfg)

	chanID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubID2, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	wrongID, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	m := senml.Message{
		Channel:    chanID,
		Publisher:  pubID,
		Protocol:   mqttProt,
		Name:       "name",
		Unit:       "U",
		UpdateTime: 1234,
	}

	messages := []senml.Message{}
	valueMsgs := []senml.Message{}
	boolMsgs := []senml.Message{}
	stringMsgs := []senml.Message{}
	dataMsgs := []senml.Message{}
	queryMsgs := []senml.Message{}
	now := float64(time.Now().Unix())

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		msg := m
		msg.Time = now - float64(i)

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

	errs := asyncWriter.Errors()
	asyncWriter.ConsumeAsync(context.TODO(), messages)
	err = <-errs
	assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	reader := ireader.New(client, repoCfg)

	cases := []struct {
		desc     string
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		{
			desc:   "read message page for existing channel",
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
		{
			desc:   "read message page for non-existent channel",
			chanID: wrongID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  msgsNum,
			},
			page: readers.MessagesPage{
				Messages: []readers.Message{},
			},
		},
		{
			desc:   "read message last page",
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
		{
			desc:   "read message with non-existent subtopic",
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
		{
			desc:   "read message with subtopic",
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
		{
			desc:   "read message with publisher",
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
		{
			desc:   "read message with wrong format",
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Format:    "messagess",
				Offset:    0,
				Limit:     uint64(len(queryMsgs)),
				Publisher: pubID2,
			},
			page: readers.MessagesPage{
				Total:    0,
				Messages: []readers.Message{},
			},
		},
		{
			desc:   "read message with protocol",
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
		{
			desc:   "read message with name",
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
		{
			desc:   "read message with value",
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				Value:  v,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(valueMsgs)),
				Messages: fromSenml(valueMsgs[0:limit]),
			},
		},
		{
			desc:   "read message with value and equal comparator",
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
		{
			desc:   "read message with value and lower-than comparator",
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
		{
			desc:   "read message with value and lower-than-or-equal comparator",
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
		{
			desc:   "read message with value and greater-than comparator",
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
		{
			desc:   "read message with value and greater-than-or-equal comparator",
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
		{
			desc:   "read message with boolean value",
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
		{
			desc:   "read message with string value",
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
		{
			desc:   "read message with data value",
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
		{
			desc:   "failing test case : read message with from",
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[0 : offset+1])),
				From:   messages[offset].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0 : offset+1])),
				Messages: fromSenml(messages[0 : offset+1]),
			},
		},
		{
			desc:   "failing test case : read message with to",
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(messages[offset-1:])),
				To:     messages[offset-1].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[offset:])),
				Messages: fromSenml(messages[offset:]),
			},
		},
		{
			desc:   "read message with from/to",
			chanID: chanID,
			pageMeta: readers.PageMetadata{
				Offset: 0,
				Limit:  limit,
				From:   messages[5].Time,
				To:     messages[0].Time,
			},
			page: readers.MessagesPage{
				Total:    uint64(len(messages[0+1 : 5+1])),
				Messages: fromSenml(messages[0+1 : 5+1]),
			},
		},
	}

	for _, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", tc.desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: got incorrect list of senml Messages from ReadAll()", tc.desc))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.page.Total, result.Total))
	}
}

func TestReadJSON(t *testing.T) {
	asyncWriter := iwriter.NewAsync(client, repoCfg)

	id1, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m := json.Message{
		Channel:   id1,
		Publisher: id1,
		Created:   time.Now().Unix() * 1e9,
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "coap",
		Payload: map[string]interface{}{
			"field_1": 123.0,
			"field_2": "value",
			"field_3": false,
		},
	}
	messages1 := json.Messages{
		Format: format1,
	}
	msgs1 := []map[string]interface{}{}
	for i := 0; i < msgsNum; i++ {
		messages1.Data = append(messages1.Data, m)
		m := toMap(m)
		msgs1 = append(msgs1, m)
	}

	errs := asyncWriter.Errors()
	asyncWriter.ConsumeAsync(context.TODO(), messages1)
	err = <-errs
	require.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	id2, err := idProvider.ID()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	m = json.Message{
		Channel:   id2,
		Publisher: id2,
		Created:   time.Now().Unix()*1e9 + msgsNum,
		Subtopic:  "subtopic/other_format/some_other_json",
		Protocol:  "udp",
		Payload: map[string]interface{}{
			"field_pi": 3.14159265,
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

	// Test async
	asyncWriter.ConsumeAsync(context.TODO(), messages2)
	err = <-errs
	assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	httpMsgs := []map[string]interface{}{}
	for i := 0; i < msgsNum; i += 2 {
		httpMsgs = append(httpMsgs, msgs2[i])
	}
	reader := ireader.New(client, repoCfg)

	cases := []struct {
		desc     string
		chanID   string
		pageMeta readers.PageMetadata
		page     readers.MessagesPage
	}{
		{
			desc:   "read message page for existing channel",
			chanID: id1,
			pageMeta: readers.PageMetadata{
				Format: messages1.Format,
				Offset: 0,
				Limit:  1,
			},
			page: readers.MessagesPage{
				Total:    msgsNum,
				Messages: fromJSON(msgs1[:1]),
			},
		},
		{
			desc:   "read message page for non-existent channel",
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
		{
			desc:   "read message last page",
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
		{
			desc:   "read message with protocol",
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

	for _, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.pageMeta)
		assert.Nil(t, err, fmt.Sprintf("%s: got unexpected error: %s", tc.desc, err))

		for i := 0; i < len(result.Messages); i++ {
			m := result.Messages[i]
			// Remove time as it is not sent by the client.
			delete(m.(map[string]interface{}), "time")

			result.Messages[i] = m
		}
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected \n%v got \n%v", tc.desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: got incorrect list of json Messages from ReadAll()", tc.desc))
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
		"subtopic":  msg.Subtopic,
		"publisher": msg.Publisher,
		"protocol":  msg.Protocol,
		"payload":   map[string]interface{}(msg.Payload),
	}
}
