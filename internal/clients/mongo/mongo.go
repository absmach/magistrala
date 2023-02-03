// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mongodb

import (
	"context"
	"fmt"

	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	errConfig  = errors.New("failed to load mongodb configuration")
	errConnect = errors.New("failed to connect to mongodb server")
)

// Config defines the options that are used when connecting to a MongoDB instance
type Config struct {
	Host string `env:"HOST" envDefault:"localhost"`
	Port string `env:"PORT" envDefault:"27017"`
	DB   string `env:"DB"   envDefault:"messages"`
}

// Connect creates a connection to the MongoDB instance
func Connect(cfg Config) (*mongo.Database, error) {
	addr := fmt.Sprintf("mongodb://%s:%s", cfg.Host, cfg.Port)
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(addr))
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}

	db := client.Database(cfg.DB)
	return db, nil
}

// Setup load configuration from environment, create new MongoDB client and connect to MongoDB server
func Setup(envPrefix string) (*mongo.Database, error) {
	cfg := Config{}
	if err := env.Parse(&cfg, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	return db, nil
}
