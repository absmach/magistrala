// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"strconv"

	r "github.com/go-redis/redis/v8"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errConfig  = errors.New("failed to load redis client configuration")
	errConnect = errors.New("failed to connect to redis server")
)

// Config of RedisDB
type Config struct {
	URL  string `env:"URL"    envDefault:"localhost:6379"`
	Pass string `env:"PASS"   envDefault:""`
	DB   string `env:"DB"     envDefault:"0"`
}

// Setup load configuration from environment, creates new RedisDB client and connect to RedisDB Server
func Setup(prefix string) (*r.Client, error) {
	cfg := Config{}
	if err := env.Parse(&cfg, env.Options{Prefix: prefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	client, err := Connect(cfg)
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	return client, nil
}

// Connect create new RedisDB client and connect to RedisDB server
func Connect(cfg Config) (*r.Client, error) {
	db, err := strconv.Atoi(cfg.DB)
	if err != nil {
		return nil, err
	}

	return r.NewClient(&r.Options{
		Addr:     cfg.URL,
		Password: cfg.Pass,
		DB:       db,
	}), nil
}
