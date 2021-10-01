// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/commands"
	log "github.com/mainflux/mainflux/logger"
)

var _ commands.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    commands.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc commands.Service, logger log.Logger) commands.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateCommand(token string, cmd commands.Command) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method CreateCommands for cmds %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateCommand(token, cmd)
}

func (lm *loggingMiddleware) ViewCommand(token, id string) (cmds commands.Command, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method ViewCommand for cmds %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewCommand(token, id)
}

func (lm *loggingMiddleware) ListCommands(token string, filter interface{}) (cmds []commands.Command, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method ListCommands for cmd %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListCommands(token, filter)
}

func (lm *loggingMiddleware) UpdateCommand(token string, cmd commands.Command) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method UpdateCommand for cmd %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateCommand(token, cmd)
}

func (lm *loggingMiddleware) RemoveCommand(token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method RemoveCommand with id %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveCommand(token, id)
}
