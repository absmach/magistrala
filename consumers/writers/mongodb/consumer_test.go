// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/absmach/magistrala/consumers/writers/mongodb"
	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	port        string
	addr        string
	testLog, _  = mglog.New(os.Stdout, mglog.Info.String())
	testDB      = "test"
	collection  = "messages"
	msgsNum     = 100
	valueFields = 5
	subtopic    = "topic"
)

var (
	v       float64 = 5
	stringV         = "value"
	boolV           = true
	dataV           = "base64"
	sum     float64 = 42
)

func TestSaveSenml(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	now := time.Now().Unix()
	msg := senml.Message{
		Channel:    "45",
		Publisher:  "2580",
		Protocol:   "http",
		Name:       "test name",
		Unit:       "km",
		Time:       13451312,
		UpdateTime: 5456565466,
	}
	var msgs []senml.Message

	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
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

		msg.Time = float64(now + int64(i))
		msgs = append(msgs, msg)
	}

	err = repo.ConsumeBlocking(context.TODO(), msgs)
	require.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	count, err := db.Collection(collection).CountDocuments(context.Background(), bson.D{})
	assert.Nil(t, err, fmt.Sprintf("Querying database expected to succeed: %s.\n", err))
	assert.Equal(t, int64(msgsNum), count, fmt.Sprintf("Expected to have %d value, found %d instead.\n", msgsNum, count))
}

func TestSaveJSON(t *testing.T) {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	chid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	pubid, err := uuid.NewV4()
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	msg := json.Message{
		Channel:   chid.String(),
		Publisher: pubid.String(),
		Created:   time.Now().Unix(),
		Subtopic:  "subtopic/format/some_json",
		Protocol:  "mqtt",
		Payload: map[string]interface{}{
			"field_1": 123,
			"field_2": "value",
			"field_3": false,
			"field_4": 12.344,
			"field_5": map[string]interface{}{
				"field_1": "value",
				"field_2": 42,
			},
		},
	}

	now := time.Now().Unix()
	msgs := json.Messages{
		Format: "some_json",
	}

	for i := 0; i < msgsNum; i++ {
		msg.Created = now + int64(i)
		msgs.Data = append(msgs.Data, msg)
	}

	err = repo.ConsumeBlocking(context.TODO(), msgs)
	assert.Nil(t, err, fmt.Sprintf("expected no error got %s\n", err))
}
