//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/bootstrap"
	log "github.com/mainflux/mainflux/logger"
)

var _ bootstrap.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    bootstrap.Service
}

// NewLoggingMiddleware adds logging facilities to the core service.
func NewLoggingMiddleware(svc bootstrap.Service, logger log.Logger) bootstrap.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Add(key string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add for key %s and thing %s took %s to complete", key, saved.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Add(key, cfg)
}

func (lm *loggingMiddleware) View(key, id string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view for key %s and thing %s took %s to complete", key, saved.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.View(key, id)
}

func (lm *loggingMiddleware) Update(key string, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update for key %s and thing %s took %s to complete", key, cfg.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Update(key, cfg)
}

func (lm *loggingMiddleware) UpdateCert(key, thingKey, clientCert, clientKey, caCert string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_cert for thing with key %s took %s to complete", thingKey, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateCert(key, thingKey, clientCert, clientKey, caCert)
}

func (lm *loggingMiddleware) UpdateConnections(key, id string, connections []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_connections for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateConnections(key, id, connections)
}

func (lm *loggingMiddleware) List(key string, filter bootstrap.Filter, offset, limit uint64) (res bootstrap.ConfigsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list for key %s and offset %d and limit %d took %s to complete", key, offset, limit, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.List(key, filter, offset, limit)
}

func (lm *loggingMiddleware) Remove(key, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Remove(key, id)
}

func (lm *loggingMiddleware) Bootstrap(externalKey, externalID string, secure bool) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method bootstrap for thing with external id %s took %s to complete", externalID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Bootstrap(externalKey, externalID, secure)
}

func (lm *loggingMiddleware) ChangeState(key, id string, state bootstrap.State) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method change_state for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ChangeState(key, id, state)
}

func (lm *loggingMiddleware) UpdateChannelHandler(channel bootstrap.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_channel_handler for channel %s took %s to complete", channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannelHandler(channel)
}

func (lm *loggingMiddleware) RemoveConfigHandler(id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_config_handler for config %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveConfigHandler(id)
}

func (lm *loggingMiddleware) RemoveChannelHandler(id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel_handler for channel %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannelHandler(id)
}

func (lm *loggingMiddleware) DisconnectThingHandler(channelID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect_thing_handler for channel %s and thing %s took %s to complete", channelID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThingHandler(channelID, thingID)
}
