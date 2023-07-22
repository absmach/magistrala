// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/messaging/brokers"
	"github.com/mainflux/mainflux/pkg/transformers"
	"github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

const (
	defContentType = "application/senml+json"
	defFormat      = "senml"
)

var (
	errOpenConfFile  = errors.New("unable to open configuration file")
	errParseConfFile = errors.New("unable to parse configuration file")
)

// Start method starts consuming messages received from Message broker.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(ctx context.Context, id string, sub messaging.Subscriber, consumer interface{}, configPath string, logger logger.Logger) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to load consumer config: %s", err))
	}

	transformer := makeTransformer(cfg.TransformerCfg, logger)

	for _, subject := range cfg.SubscriberCfg.Subjects {
		switch c := consumer.(type) {
		case AsyncConsumer:
			if err := sub.Subscribe(ctx, id, subject, handleAsync(ctx, transformer, c)); err != nil {
				return err
			}
		case BlockingConsumer:
			if err := sub.Subscribe(ctx, id, subject, handleSync(ctx, transformer, c)); err != nil {
				return err
			}
		default:
			return errors.ErrInvalidQueryParams
		}

	}
	return nil
}

func handleSync(ctx context.Context, t transformers.Transformer, sc BlockingConsumer) handleFunc {
	return func(msg *messaging.Message) error {
		m := interface{}(msg)
		var err error
		if t != nil {
			m, err = t.Transform(msg)
			if err != nil {
				return err
			}
		}
		return sc.ConsumeBlocking(ctx, m)
	}
}

func handleAsync(ctx context.Context, t transformers.Transformer, ac AsyncConsumer) handleFunc {
	return func(msg *messaging.Message) error {
		m := interface{}(msg)
		var err error
		if t != nil {
			m, err = t.Transform(msg)
			if err != nil {
				return err
			}
		}

		ac.ConsumeAsync(ctx, m)
		return nil
	}
}

type handleFunc func(msg *messaging.Message) error

func (h handleFunc) Handle(msg *messaging.Message) error {
	return h(msg)
}

func (h handleFunc) Cancel() error {
	return nil
}

type subscriberConfig struct {
	Subjects []string `toml:"subjects"`
}

type transformerConfig struct {
	Format      string           `toml:"format"`
	ContentType string           `toml:"content_type"`
	TimeFields  []json.TimeField `toml:"time_fields"`
}

type config struct {
	SubscriberCfg  subscriberConfig  `toml:"subscriber"`
	TransformerCfg transformerConfig `toml:"transformer"`
}

func loadConfig(configPath string) (config, error) {
	cfg := config{
		SubscriberCfg: subscriberConfig{
			Subjects: []string{brokers.SubjectAllChannels},
		},
		TransformerCfg: transformerConfig{
			Format:      defFormat,
			ContentType: defContentType,
		},
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, errors.Wrap(errOpenConfFile, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, errors.Wrap(errParseConfFile, err)
	}

	return cfg, nil
}

func makeTransformer(cfg transformerConfig, logger logger.Logger) transformers.Transformer {
	switch strings.ToUpper(cfg.Format) {
	case "SENML":
		logger.Info("Using SenML transformer")
		return senml.New(cfg.ContentType)
	case "JSON":
		logger.Info("Using JSON transformer")
		return json.New(cfg.TimeFields)
	default:
		logger.Error(fmt.Sprintf("Can't create transformer: unknown transformer type %s", cfg.Format))
		os.Exit(1)
		return nil
	}
}
