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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writers/mongodb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mongodb/mongo-go-driver/mongo"
)

var (
	port        string
	addr        string
	testLog, _  = log.New(os.Stdout, log.Info.String())
	testDB      = "test"
	collection  = "mainflux"
	db          mongo.Database
	msgsNum     = 100
	valueFields = 6
)

func TestSave(t *testing.T) {
	msg := mainflux.Message{
		Channel:    "45",
		Publisher:  "2580",
		Protocol:   "http",
		Name:       "test name",
		Unit:       "km",
		Value:      &mainflux.Message_FloatValue{24},
		ValueSum:   &mainflux.SumValue{Value: 24},
		Time:       13451312,
		UpdateTime: 5456565466,
		Link:       "link",
	}

	client, err := mongo.Connect(context.Background(), addr, nil)
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	now := time.Now().Unix()
	for i := 0; i < msgsNum; i++ {
		// Mix possible values as well as value sum.
		count := i % valueFields
		switch count {
		case 0:
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
		msg.Time = float64(now + int64(i))

		err = repo.Save(msg)
	}
	assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	count, err := db.Collection(collection).Count(context.Background(), nil)
	assert.Nil(t, err, fmt.Sprintf("Querying database expected to succeed: %s.\n", err))
	assert.Equal(t, int64(msgsNum), count, fmt.Sprintf("Expected to have %d value, found %d instead.\n", msgsNum, count))
}
