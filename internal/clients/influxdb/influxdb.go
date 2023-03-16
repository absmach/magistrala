package influxdb

import (
	"context"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	errConnect = errors.New("failed to create InfluxDB client")
	errConfig  = errors.New("failed to load InfluxDB client configuration from environment variable")
)

type Config struct {
	Protocol string        `env:"PROTOCOL"              envDefault:"http"`
	Host     string        `env:"HOST"                  envDefault:"localhost"`
	Port     string        `env:"PORT"                  envDefault:"8086"`
	Bucket   string        `env:"BUCKET"                envDefault:"mainflux-bucket"`
	Org      string        `env:"ORG"                   envDefault:"mainflux"`
	Token    string        `env:"TOKEN"                 envDefault:"mainflux-token"`
	DBUrl    string        `env:"DBURL"                 envDefault:""`
	Timeout  time.Duration `env:"TIMEOUT"               envDefault:"1s"`
}

// Setup load configuration from environment variable, create InfluxDB client and connect to InfluxDB server
func Setup(envPrefix string, ctx context.Context) (influxdb2.Client, error) {
	config := Config{}
	if err := env.Parse(&config, env.Options{Prefix: envPrefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	return Connect(config, ctx)
}

// Connect create InfluxDB client and connect to InfluxDB server
func Connect(config Config, ctx context.Context) (influxdb2.Client, error) {
	client := influxdb2.NewClient(config.DBUrl, config.Token)
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	if _, err := client.Ready(ctx); err != nil {
		return nil, errors.Wrap(errConnect, err)
	}
	return client, nil
}
