package influxdb_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux/readers"
	reader "github.com/mainflux/mainflux/readers/influxdb"
	"github.com/mainflux/mainflux/transformers/senml"
	writer "github.com/mainflux/mainflux/writers/influxdb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDB   = "test"
	chanID   = "1"
	subtopic = "topic"
	msgsNum  = 101
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

var (
	valueFields = 5
	port        string
	client      influxdata.Client
	testLog, _  = log.New(os.Stdout, log.Info.String())

	clientCfg = influxdata.HTTPConfig{
		Username: "test",
		Password: "test",
	}

	m = senml.Message{
		Channel:    chanID,
		Publisher:  "1",
		Protocol:   "mqtt",
		Name:       "name",
		Unit:       "U",
		Time:       123456,
		UpdateTime: 1234,
	}
)

func TestReadAll(t *testing.T) {
	writer := writer.New(client, testDB)

	messages := []senml.Message{}
	subtopicMsgs := []senml.Message{}
	now := time.Now().UnixNano()
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

		msg.Time = float64(now)/float64(1e9) - float64(i)
		messages = append(messages, msg)
		if count == 0 {
			subtopicMsgs = append(subtopicMsgs, msg)
		}
	}

	err := writer.Save(messages...)
	require.Nil(t, err, fmt.Sprintf("failed to store message to InfluxDB: %s", err))

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
		"read message page for non-existent channel": {
			chanID: "2",
			offset: 0,
			limit:  10,
			page: readers.MessagesPage{
				Total:    0,
				Offset:   0,
				Limit:    10,
				Messages: []senml.Message{},
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
				Messages: []senml.Message{},
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
