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

// New returns new MongoDB writer.
func New(db *mongo.Database) writers.MessageRepository {
	return &mongoRepo{db}
}

func (repo *mongoRepo) Save(msg mainflux.Message) error {
	coll := repo.db.Collection(collectionName)
	_, err := coll.InsertOne(context.Background(), msg)
	return err
}
