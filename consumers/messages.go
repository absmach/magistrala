// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package consumers

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"
	pubsub "github.com/mainflux/mainflux/pkg/messaging/nats"
	"github.com/mainflux/mainflux/pkg/transformers"
)

var (
	errOpenConfFile  = errors.New("unable to open configuration file")
	errParseConfFile = errors.New("unable to parse configuration file")
)

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(sub messaging.Subscriber, consumer Consumer, transformer transformers.Transformer, subjectsCfgPath string, logger logger.Logger) error {
	subjects, err := loadSubjectsConfig(subjectsCfgPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to load subjects: %s", err))
	}

	for _, subject := range subjects {
		if err := sub.Subscribe(subject, handler(transformer, consumer)); err != nil {
			return err
		}
	}
	return nil
}

func handler(t transformers.Transformer, c Consumer) messaging.MessageHandler {
	return func(msg messaging.Message) error {
		m := interface{}(msg)
		var err error
		if t != nil {
			m, err = t.Transform(msg)
			if err != nil {
				return err
			}
		}
		return c.Consume(m)
	}
}

type filterConfig struct {
	Filter []string `toml:"filter"`
}

type subjectsConfig struct {
	Subjects filterConfig `toml:"subjects"`
}

func loadSubjectsConfig(subjectsConfigPath string) ([]string, error) {
	data, err := ioutil.ReadFile(subjectsConfigPath)
	if err != nil {
		return []string{pubsub.SubjectAllChannels}, errors.Wrap(errOpenConfFile, err)
	}

	var subjectsCfg subjectsConfig
	if err := toml.Unmarshal(data, &subjectsCfg); err != nil {
		return []string{pubsub.SubjectAllChannels}, errors.Wrap(errParseConfFile, err)
	}

	return subjectsCfg.Subjects.Filter, nil
}
