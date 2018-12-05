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

// Message struct is used as a MongoDB representation of Mainflux message.
type message struct {
	Channel     string   `bson:"channel,omitempty"`
	Publisher   string   `bson:"publisher,omitempty"`
	Protocol    string   `bson:"protocol,omitempty"`
	Name        string   `bson:"name,omitempty"`
	Unit        string   `bson:"unit,omitempty"`
	FloatValue  *float64 `bson:"value,omitempty"`
	StringValue *string  `bson:"stringValue,omitempty"`
	BoolValue   *bool    `bson:"boolValue,omitempty"`
	DataValue   *string  `bson:"dataValue,omitempty"`
	ValueSum    *float64 `bson:"valueSum,omitempty"`
	Time        float64  `bson:"time,omitempty"`
	UpdateTime  float64  `bson:"updateTime,omitempty"`
	Link        string   `bson:"link,omitempty"`
}

// New returns new MongoDB reader.
func New(db *mongo.Database) readers.MessageRepository {
	return mongoRepository{db: db}
}

func (repo mongoRepository) ReadAll(chanID string, offset, limit uint64) []mainflux.Message {
	col := repo.db.Collection(collection)
	cursor, err := col.Find(context.Background(), bson.NewDocument(bson.EC.String("channel", chanID)), findopt.Limit(int64(limit)), findopt.Skip(int64(offset)))
	if err != nil {
		return []mainflux.Message{}
	}
	defer cursor.Close(context.Background())

	messages := []mainflux.Message{}
	for cursor.Next(context.Background()) {
		var m message
		if err := cursor.Decode(&m); err != nil {
			return []mainflux.Message{}
		}

		msg := mainflux.Message{
			Channel:    m.Channel,
			Publisher:  m.Publisher,
			Protocol:   m.Protocol,
			Name:       m.Name,
			Unit:       m.Unit,
			Time:       m.Time,
			UpdateTime: m.UpdateTime,
			Link:       m.Link,
		}

		switch {
		case m.FloatValue != nil:
			msg.Value = &mainflux.Message_FloatValue{FloatValue: *m.FloatValue}
		case m.StringValue != nil:
			msg.Value = &mainflux.Message_StringValue{StringValue: *m.StringValue}
		case m.DataValue != nil:
			msg.Value = &mainflux.Message_DataValue{DataValue: *m.DataValue}
		case m.BoolValue != nil:
			msg.Value = &mainflux.Message_BoolValue{BoolValue: *m.BoolValue}
		}

		if m.ValueSum != nil {
			msg.ValueSum = &mainflux.SumValue{Value: *m.ValueSum}
		}

		messages = append(messages, msg)
	}

	return messages
}
