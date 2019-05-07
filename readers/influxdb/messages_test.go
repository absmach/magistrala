package influxdb_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/influxdb"
	writer "github.com/mainflux/mainflux/writers/influxdb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDB      = "test"
	chanID      = "1"
	subtopic    = "topic"
	msgsNum     = 101
	valueFields = 6
)

var (
	port       string
	client     influxdata.Client
	testLog, _ = log.New(os.Stdout, log.Info.String())

	clientCfg = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}

	msg = mainflux.Message{
		Channel:    chanID,
		Publisher:  "1",
		Protocol:   "mqtt",
		Name:       "name",
		Unit:       "U",
		Value:      &mainflux.Message_FloatValue{FloatValue: 5},
		ValueSum:   &mainflux.SumValue{Value: 45},
		Time:       123456,
		UpdateTime: 1234,
		Link:       "link",
	}
)

func TestReadAll(t *testing.T) {
	writer, err := writer.New(client, testDB, 1, time.Second)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB writer expected to succeed: %s.\n", err))

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
		require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))
		messages = append(messages, msg)
		if count == 0 {
			subtopicMsgs = append(subtopicMsgs, msg)
		}
	}

	reader := reader.New(client, testDB)
	require.Nil(t, err, fmt.Sprintf("Creating new InfluxDB reader expected to succeed: %s.\n", err))

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
		"read message page for too large limit": {
			chanID: chanID,
			offset: 0,
			limit:  101,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   0,
				Limit:    101,
				Messages: messages[0:100],
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
			offset: 95,
			limit:  10,
			page: readers.MessagesPage{
				Total:    msgsNum,
				Offset:   95,
				Limit:    10,
				Messages: messages[95:101],
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
				Messages: subtopicMsgs[0:10],
			},
		},
	}

	for desc, tc := range cases {
		result, err := reader.ReadAll(tc.chanID, tc.offset, tc.limit, tc.query)
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %s", desc, err))
		assert.ElementsMatch(t, tc.page.Messages, result.Messages, fmt.Sprintf("%s: expected: %v \n-------------\n got: %v", desc, tc.page.Messages, result.Messages))
		assert.Equal(t, tc.page.Total, result.Total, fmt.Sprintf("%s: expected %d got %d", desc, tc.page.Total, result.Total))
	}
}
