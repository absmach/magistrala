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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writers/mongodb"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mongodb/mongo-go-driver/mongo"
)

var (
	port       string
	addr       string
	testLog, _ = log.New(os.Stdout, log.Info.String())
	testDB     = "test"
	collection = "mainflux"
	db         mongo.Database
)

func TestSave(t *testing.T) {
	msg := mainflux.Message{
		Channel:     45,
		Publisher:   2580,
		Protocol:    "http",
		Name:        "test name",
		Unit:        "km",
		Value:       24,
		StringValue: "24",
		BoolValue:   false,
		DataValue:   "dataValue",
		ValueSum:    24,
		Time:        13451312,
		UpdateTime:  5456565466,
		Link:        "link",
	}

	client, err := mongo.Connect(context.Background(), addr, nil)
	require.Nil(t, err, fmt.Sprintf("Creating new MongoDB client expected to succeed: %s.\n", err))

	db := client.Database(testDB)
	repo := mongodb.New(db)

	err = repo.Save(msg)
	assert.Nil(t, err, fmt.Sprintf("Save operation expected to succeed: %s.\n", err))

	count, err := db.Collection(collection).Count(context.Background(), nil)
	assert.Nil(t, err, fmt.Sprintf("Querying database expected to succeed: %s.\n", err))
	assert.Equal(t, int64(1), count, fmt.Sprintf("Expected to have 1 value, found %d instead.\n", count))
}
