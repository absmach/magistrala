// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
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

func (lm *loggingMiddleware) Add(ctx context.Context, token string, cfg bootstrap.Config) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add for token %s and thing %s took %s to complete", token, saved.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Add(ctx, token, cfg)
}

func (lm *loggingMiddleware) View(ctx context.Context, token, id string) (saved bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view for token %s and thing %s took %s to complete", token, saved.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.View(ctx, token, id)
}

func (lm *loggingMiddleware) Update(ctx context.Context, token string, cfg bootstrap.Config) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update for token %s and thing %s took %s to complete", token, cfg.MFThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Update(ctx, token, cfg)
}

func (lm *loggingMiddleware) UpdateCert(ctx context.Context, token, thingID, clientCert, clientKey, caCert string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_cert for thing with id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateCert(ctx, token, thingID, clientCert, clientKey, caCert)
}

func (lm *loggingMiddleware) UpdateConnections(ctx context.Context, token, id string, connections []string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_connections for token %s and thing %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateConnections(ctx, token, id, connections)
}

func (lm *loggingMiddleware) List(ctx context.Context, token string, filter bootstrap.Filter, offset, limit uint64) (res bootstrap.ConfigsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list for token %s and offset %d and limit %d took %s to complete", token, offset, limit, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.List(ctx, token, filter, offset, limit)
}

func (lm *loggingMiddleware) Remove(ctx context.Context, token, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove for token %s and thing %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Remove(ctx, token, id)
}

func (lm *loggingMiddleware) Bootstrap(ctx context.Context, externalKey, externalID string, secure bool) (cfg bootstrap.Config, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method bootstrap for thing with external id %s took %s to complete", externalID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Bootstrap(ctx, externalKey, externalID, secure)
}

func (lm *loggingMiddleware) ChangeState(ctx context.Context, token, id string, state bootstrap.State) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method change_state for token %s and thing %s took %s to complete", token, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ChangeState(ctx, token, id, state)
}

func (lm *loggingMiddleware) UpdateChannelHandler(ctx context.Context, channel bootstrap.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_channel_handler for channel %s took %s to complete", channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannelHandler(ctx, channel)
}

func (lm *loggingMiddleware) RemoveConfigHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_config_handler for config %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveConfigHandler(ctx, id)
}

func (lm *loggingMiddleware) RemoveChannelHandler(ctx context.Context, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel_handler for channel %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannelHandler(ctx, id)
}

func (lm *loggingMiddleware) DisconnectThingHandler(ctx context.Context, channelID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect_thing_handler for channel %s and thing %s took %s to complete", channelID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThingHandler(ctx, channelID, thingID)
}
