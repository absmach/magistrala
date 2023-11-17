// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"

	"github.com/absmach/magistrala/consumers"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const senmlCollection string = "messages"

var errSaveMessage = errors.New("failed to save message to mongodb database")

var _ consumers.BlockingConsumer = (*mongoRepo)(nil)

type mongoRepo struct {
	db *mongo.Database
}

// New returns new MongoDB writer.
func New(db *mongo.Database) consumers.BlockingConsumer {
	return &mongoRepo{db}
}

func (repo *mongoRepo) ConsumeBlocking(ctx context.Context, message interface{}) error {
	switch m := message.(type) {
	case json.Messages:
		return repo.saveJSON(ctx, m)
	default:
		return repo.saveSenml(ctx, m)
	}
}

func (repo *mongoRepo) saveSenml(ctx context.Context, messages interface{}) error {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	coll := repo.db.Collection(senmlCollection)
	var dbMsgs []interface{}
	for _, msg := range msgs {
		// Check if message is already in database.
		filter := bson.M{"time": msg.Time, "publisher": msg.Publisher, "subtopic": msg.Subtopic, "name": msg.Name}

		count, err := coll.CountDocuments(ctx, filter)
		if err != nil {
			return errors.Wrap(errSaveMessage, err)
		}

		if count == 0 {
			dbMsgs = append(dbMsgs, msg)
		}
	}

	_, err := coll.InsertMany(ctx, dbMsgs)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}

	return nil
}

func (repo *mongoRepo) saveJSON(ctx context.Context, msgs json.Messages) error {
	m := []interface{}{}
	for _, msg := range msgs.Data {
		m = append(m, msg)
	}

	coll := repo.db.Collection(msgs.Format)

	_, err := coll.InsertMany(ctx, m)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}

	return nil
}
