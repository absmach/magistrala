// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/opcua"
)

var _ opcua.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    opcua.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc opcua.Service, logger logger.Logger) opcua.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) CreateThing(mfxThing string, opcID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing mfx:opcua:%s:%s took %s to complete", mfxThing, opcID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(mfxThing, opcID)
}

func (lm loggingMiddleware) UpdateThing(mfxThing string, opcID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing mfx:opcua:%s:%s took %s to complete", mfxThing, opcID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(mfxThing, opcID)
}

func (lm loggingMiddleware) RemoveThing(mfxThing string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing mfx:opcua:%s took %s to complete", mfxThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(mfxThing)
}

func (lm loggingMiddleware) CreateChannel(mfxChan string, opcNamespace string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel mfx:opcua:%s:%s took %s to complete", mfxChan, opcNamespace, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(mfxChan, opcNamespace)
}

func (lm loggingMiddleware) UpdateChannel(mfxChanID string, opcNamespace string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel mfx:opcua:%s:%s took %s to complete", mfxChanID, opcNamespace, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(mfxChanID, opcNamespace)
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

func (lm loggingMiddleware) Publish(ctx context.Context, token string, m opcua.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("publish namespace/%s/id/%s/rx took %s to complete", m.Namespace, m.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, token, m)
}
