// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package writers

import (
	"fmt"
	"io/ioutil"

	"github.com/BurntSushi/toml"
	"github.com/gogo/protobuf/proto"
	"github.com/mainflux/mainflux/broker"
	"github.com/mainflux/mainflux/errors"
	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/transformers"
	"github.com/mainflux/mainflux/transformers/senml"
	"github.com/nats-io/nats.go"
)

var (
	errOpenConfFile  = errors.New("Unable to open configuration file")
	errParseConfFile = errors.New("Unable to parse configuration file")
)

type consumer struct {
	broker      broker.Nats
	repo        MessageRepository
	transformer transformers.Transformer
	logger      logger.Logger
}

// Start method starts consuming messages received from NATS.
// This method transforms messages to SenML format before
// using MessageRepository to store them.
func Start(broker broker.Nats, repo MessageRepository, transformer transformers.Transformer, queue string, subjectsCfgPath string, logger logger.Logger) error {
	c := consumer{
		broker:      broker,
		repo:        repo,
		transformer: transformer,
		logger:      logger,
	}

	subjects, err := loadSubjectsConfig(subjectsCfgPath)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to load subjects: %s", err))
	}

	for _, subject := range subjects {
		_, err := broker.QueueSubscribe(subject, queue, c.consume)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *consumer) consume(m *nats.Msg) {
	var msg broker.Message
	if err := proto.Unmarshal(m.Data, &msg); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to unmarshal received message: %s", err))
		return
	}

	t, err := c.transformer.Transform(msg)
	if err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to tranform received message: %s", err))
		return
	}
	msgs, ok := t.([]senml.Message)
	if !ok {
		c.logger.Warn("Invalid message format from the Transformer output.")
		return
	}

	if err := c.repo.Save(msgs...); err != nil {
		c.logger.Warn(fmt.Sprintf("Failed to save message: %s", err))
		return
	}
}

type filterConfig struct {
	List []string `toml:"filter"`
}

type subjectsConfig struct {
	Subjects filterConfig `toml:"subjects"`
}

func loadSubjectsConfig(subjectsConfigPath string) ([]string, error) {
	data, err := ioutil.ReadFile(subjectsConfigPath)
	if err != nil {
		return []string{broker.SubjectAllChannels}, errors.Wrap(errOpenConfFile, err)
	}

	var subjectsCfg subjectsConfig
	if err := toml.Unmarshal(data, &subjectsCfg); err != nil {
		return []string{broker.SubjectAllChannels}, errors.Wrap(errParseConfFile, err)
	}

	return subjectsCfg.Subjects.List, nil
}
