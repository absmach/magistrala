package cassandra_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux"
	readers "github.com/mainflux/mainflux/readers/cassandra"
	writers "github.com/mainflux/mainflux/writers/cassandra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	keyspace      = "mainflux"
	chanID        = 1
	numOfMessages = 42
)

var (
	addr = "localhost"
	msg  = mainflux.Message{
		Channel:   chanID,
		Publisher: 1,
		Protocol:  "mqtt",
	}
)

func TestReadAll(t *testing.T) {
	session, err := readers.Connect([]string{addr}, keyspace)
	require.Nil(t, err, fmt.Sprintf("failed to connect to Cassandra: %s", err))
	defer session.Close()
	writer := writers.New(session)

	messages := []mainflux.Message{}
	for i := 0; i < numOfMessages; i++ {
		err := writer.Save(msg)
		require.Nil(t, err, fmt.Sprintf("failed to store message to Cassandra: %s", err))
		messages = append(messages, msg)
	}

	reader := readers.New(session)

	cases := map[string]struct {
		chanID   uint64
		offset   uint64
		limit    uint64
		messages []mainflux.Message
	}{
		"read message page for existing channel": {
			chanID:   chanID,
			offset:   0,
			limit:    10,
			messages: messages[0:10],
		},
		"read message page for non-existent channel": {
			chanID:   2,
			offset:   0,
			limit:    10,
			messages: []mainflux.Message{},
		},
		"read message last page": {
			chanID:   chanID,
			offset:   40,
			limit:    10,
			messages: messages[40:42],
		},
	}

	for desc, tc := range cases {
		result := reader.ReadAll(tc.chanID, tc.offset, tc.limit)
		assert.ElementsMatch(t, tc.messages, result, fmt.Sprintf("%s: expected %v got %v", desc, tc.messages, result))
	}
}
