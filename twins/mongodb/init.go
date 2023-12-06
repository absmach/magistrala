// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"fmt"

	mglog "github.com/absmach/magistrala/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config defines the options that are used when connecting to a MongoDB instance.
type Config struct {
	Host string
	Port string
	Name string
}

// Connect creates a connection to the MongoDB instance.
func Connect(cfg Config, logger mglog.Logger) (*mongo.Database, error) {
	addr := fmt.Sprintf("mongodb://%s:%s", cfg.Host, cfg.Port)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to connect to database: %s", err))
		return nil, err
	}

	db := client.Database(cfg.Name)
	return db, nil
}
