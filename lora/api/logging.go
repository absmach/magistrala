// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/lora"
)

var _ lora.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    lora.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc lora.Service, logger logger.Logger) lora.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) CreateThing(thingID string, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing for thing %s and lora-dev-eui %s took %s to complete", thingID, loraDevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(thingID, loraDevEUI)
}

func (lm loggingMiddleware) UpdateThing(thingID string, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing for thing %s and lora-dev-eui %s took %s to complete", thingID, loraDevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(thingID, loraDevEUI)
}

func (lm loggingMiddleware) RemoveThing(thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(thingID)
}

func (lm loggingMiddleware) CreateChannel(chanID string, loraApp string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel for channel %s and lora-app %s took %s to complete", chanID, loraApp, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(chanID, loraApp)
}

func (lm loggingMiddleware) UpdateChannel(chanID string, loraApp string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel for channel %s and lora-app %s took %s to complete", chanID, loraApp, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(chanID, loraApp)
}

func (lm loggingMiddleware) RemoveChannel(chanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_channel for channel %s took %s to complete", chanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(chanID)
}

func (lm loggingMiddleware) ConnectThing(chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("connect_thing for channel %s and thing %s, took %s to complete", chanID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConnectThing(chanID, thingID)
}

func (lm loggingMiddleware) DisconnectThing(chanID, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("disconnect_thing mfx-%s : mfx-%s, took %s to complete", chanID, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThing(chanID, thingID)
}

func (lm loggingMiddleware) Publish(m lora.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("publish application/%s/device/%s/rx took %s to complete", m.ApplicationID, m.DevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(m)
}
