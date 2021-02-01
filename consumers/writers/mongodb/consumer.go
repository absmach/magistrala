// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mainflux/mainflux/consumers"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

const senmlCollection string = "messages"

var errSaveMessage = errors.New("failed to save message to mongodb database")

var _ consumers.Consumer = (*mongoRepo)(nil)

type mongoRepo struct {
	db *mongo.Database
}

// New returns new MongoDB writer.
func New(db *mongo.Database) consumers.Consumer {
	return &mongoRepo{db}
}

func (repo *mongoRepo) Consume(message interface{}) error {
	switch m := message.(type) {
	case json.Messages:
		return repo.saveJSON(m)
	default:
		return repo.saveSenml(m)
	}

}

func (repo *mongoRepo) saveSenml(messages interface{}) error {
	msgs, ok := messages.([]senml.Message)
	if !ok {
		return errSaveMessage
	}
	coll := repo.db.Collection(senmlCollection)
	var dbMsgs []interface{}
	for _, msg := range msgs {
		dbMsgs = append(dbMsgs, msg)
	}

	_, err := coll.InsertMany(context.Background(), dbMsgs)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}

	return nil
}

func (repo *mongoRepo) saveJSON(msgs json.Messages) error {
	m := []interface{}{}
	for _, msg := range msgs.Data {
		m = append(m, msg)
	}

	coll := repo.db.Collection(msgs.Format)

	_, err := coll.InsertMany(context.Background(), m)
	if err != nil {
		return errors.Wrap(errSaveMessage, err)
	}

	return nil
}
