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

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
)

var _ things.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    things.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc things.Service, logger log.Logger) things.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddThing(key string, thing things.Thing) (saved things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_thing for key %s and thing %s took %s to complete", key, saved.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddThing(key, thing)
}

func (lm *loggingMiddleware) UpdateThing(key string, thing things.Thing) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_thing for key %s and thing %s took %s to complete", key, thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(key, thing)
}

func (lm *loggingMiddleware) ViewThing(key, id string) (thing things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_thing for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewThing(key, id)
}

func (lm *loggingMiddleware) ListThings(key string, offset, limit uint64) (things []things.Thing, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_things for key %s took %s to complete", key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListThings(key, offset, limit)
}

func (lm *loggingMiddleware) RemoveThing(key, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_thing for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(key, id)
}

func (lm *loggingMiddleware) CreateChannel(key string, channel things.Channel) (saved things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_channel for key %s and channel %s took %s to complete", key, channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(key, channel)
}

func (lm *loggingMiddleware) UpdateChannel(key string, channel things.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_channel for key %s and channel %s took %s to complete", key, channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(key, channel)
}

func (lm *loggingMiddleware) ViewChannel(key, id string) (channel things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel for key %s and channel %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannel(key, id)
}

func (lm *loggingMiddleware) ListChannels(key string, offset, limit uint64) (channels []things.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_channels for key %s took %s to complete", key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChannels(key, offset, limit)
}

func (lm *loggingMiddleware) RemoveChannel(key, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel for key %s and channel %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(key, id)
}

func (lm *loggingMiddleware) Connect(key, chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect for key %s, channel %s and thing %s took %s to complete", key, chanID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(key, chanID, thingID)
}

func (lm *loggingMiddleware) Disconnect(key, chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect for key %s, channel %s and thing %s took %s to complete", key, chanID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(key, chanID, thingID)
}

func (lm *loggingMiddleware) CanAccess(id, key string) (thing string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access for channel %s and thing %s took %s to complete", id, thing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanAccess(id, key)
}

func (lm *loggingMiddleware) Identify(key string) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method identify for key %s and thing %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Identify(key)
}
