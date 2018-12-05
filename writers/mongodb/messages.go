//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package mongodb

import (
	"context"

	"github.com/mongodb/mongo-go-driver/mongo"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/writers"
)

const collectionName string = "mainflux"

var _ writers.MessageRepository = (*mongoRepo)(nil)

type mongoRepo struct {
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

// New returns new MongoDB writer.
func New(db *mongo.Database) writers.MessageRepository {
	return &mongoRepo{db}
}

func (repo *mongoRepo) Save(msg mainflux.Message) error {
	coll := repo.db.Collection(collectionName)
	m := message{
		Channel:    msg.Channel,
		Publisher:  msg.Publisher,
		Protocol:   msg.Protocol,
		Name:       msg.Name,
		Unit:       msg.Unit,
		Time:       msg.Time,
		UpdateTime: msg.UpdateTime,
		Link:       msg.Link,
	}

	switch msg.Value.(type) {
	case *mainflux.Message_FloatValue:
		v := msg.GetFloatValue()
		m.FloatValue = &v
	case *mainflux.Message_StringValue:
		v := msg.GetStringValue()
		m.StringValue = &v
	case *mainflux.Message_DataValue:
		v := msg.GetDataValue()
		m.DataValue = &v
	case *mainflux.Message_BoolValue:
		v := msg.GetBoolValue()
		m.BoolValue = &v
	}

	if msg.GetValueSum() != nil {
		valueSum := msg.GetValueSum().Value
		m.ValueSum = &valueSum
	}

	_, err := coll.InsertOne(context.Background(), m)
	return err
}
