//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"
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

func (lm loggingMiddleware) CreateThing(mfxThing string, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing mfx:lora:%s:%s took %s to complete", mfxThing, loraDevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(mfxThing, loraDevEUI)
}

func (lm loggingMiddleware) UpdateThing(mfxThing string, loraDevEUI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing mfx:lora:%s:%s took %s to complete", mfxThing, loraDevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(mfxThing, loraDevEUI)
}

func (lm loggingMiddleware) RemoveThing(mfxThing string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing mfx:lora:%s took %s to complete", mfxThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(mfxThing)
}

func (lm loggingMiddleware) CreateChannel(mfxChan string, loraApp string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel mfx:lora:%s:%s took %s to complete", mfxChan, loraApp, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(mfxChan, loraApp)
}

func (lm loggingMiddleware) UpdateChannel(mfxChanID string, loraApp string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel mfx:lora:%s:%s took %s to complete", mfxChanID, loraApp, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(mfxChanID, loraApp)
}

func (lm loggingMiddleware) RemoveChannel(mfxChanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_channel mfx_channel_%s took %s to complete", mfxChanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(mfxChanID)
}

func (lm loggingMiddleware) Publish(ctx context.Context, token string, m lora.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("message_router application/%s/device/%s/rx took %s to complete", m.ApplicationID, m.DevEUI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, token, m)
}
