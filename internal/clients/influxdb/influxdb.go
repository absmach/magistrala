// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"context"
	"time"

	"github.com/absmach/magistrala/pkg/errors"
	"github.com/caarlos0/env/v10"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

var (
	errConnect = errors.New("failed to create InfluxDB client")
	errConfig  = errors.New("failed to load InfluxDB client configuration from environment variable")
)

type Config struct {
	Protocol           string        `env:"PROTOCOL"              envDefault:"http"`
	Host               string        `env:"HOST"                  envDefault:"localhost"`
	Port               string        `env:"PORT"                  envDefault:"8086"`
	Username           string        `env:"ADMIN_USER"            envDefault:"magistrala"`
	Password           string        `env:"ADMIN_PASSWORD"        envDefault:"magistrala"`
	DBName             string        `env:"NAME"                  envDefault:"magistrala"`
	Bucket             string        `env:"BUCKET"                envDefault:"magistrala-bucket"`
	Org                string        `env:"ORG"                   envDefault:"magistrala"`
	Token              string        `env:"TOKEN"                 envDefault:"magistrala-token"`
	DBUrl              string        `env:"DBURL"                 envDefault:""`
	UserAgent          string        `env:"USER_AGENT"            envDefault:"InfluxDBClient"`
	Timeout            time.Duration `env:"TIMEOUT"` // Influxdb client configuration by default has no timeout duration , this field will not have a fallback default timeout duration. Reference: https://pkg.go.dev/github.com/influxdata/influxdb@v1.10.0/client/v2#HTTPConfig
	InsecureSkipVerify bool          `env:"INSECURE_SKIP_VERIFY"  envDefault:"false"`
}

// Setup load configuration from environment variable, create InfluxDB client and connect to InfluxDB server.
func Setup(ctx context.Context, envPrefix string) (influxdb2.Client, error) {
	cfg := Config{}
	if err := env.ParseWithOptions(&cfg, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	return Connect(ctx, cfg)
}

// Connect create InfluxDB client and connect to InfluxDB server.
func Connect(ctx context.Context, config Config) (influxdb2.Client, error) {
	client := influxdb2.NewClientWithOptions(config.DBUrl, config.Token,
		influxdb2.DefaultOptions().
			SetUseGZip(true).
			SetFlushInterval(100))
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	if _, err := client.Ready(ctx); err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	return client, nil
}
