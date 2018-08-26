//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mongodb

import (
	"context"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/readers"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
)

const collection = "mainflux"

var _ readers.MessageRepository = (*mongoRepository)(nil)

type mongoRepository struct {
	db *mongo.Database
}

// New returns new MongoDB reader.
func New(db *mongo.Database) readers.MessageRepository {
	return mongoRepository{db: db}
}

func (repo mongoRepository) ReadAll(chanID, offset, limit uint64) []mainflux.Message {
	col := repo.db.Collection(collection)
	cursor, err := col.Find(context.Background(), bson.NewDocument(bson.EC.Int64("channel", int64(chanID))), findopt.Limit(int64(limit)), findopt.Skip(int64(offset)))
	if err != nil {
		return []mainflux.Message{}
	}
	defer cursor.Close(context.Background())

	messages := []mainflux.Message{}
	for cursor.Next(context.Background()) {
		var msg mainflux.Message
		if err := cursor.Decode(&msg); err != nil {
			return []mainflux.Message{}
		}

		messages = append(messages, msg)
	}

	return messages
}
