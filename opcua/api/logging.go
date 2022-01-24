// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

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

func (lm loggingMiddleware) CreateThing(ctx context.Context, mfxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing %s with NodeID %s, took %s to complete", mfxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(ctx, mfxThing, opcuaNodeID)
}

func (lm loggingMiddleware) UpdateThing(ctx context.Context, mfxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing %s with NodeID %s, took %s to complete", mfxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(ctx, mfxThing, opcuaNodeID)
}

func (lm loggingMiddleware) RemoveThing(ctx context.Context, mfxThing string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing %s, took %s to complete", mfxThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(ctx, mfxThing)
}

func (lm loggingMiddleware) CreateChannel(ctx context.Context, mfxChan, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel %s with ServerURI %s, took %s to complete", mfxChan, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(ctx, mfxChan, opcuaServerURI)
}

func (lm loggingMiddleware) UpdateChannel(ctx context.Context, mfxChanID, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel %s with ServerURI %s, took %s to complete", mfxChanID, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(ctx, mfxChanID, opcuaServerURI)
}

func (lm loggingMiddleware) RemoveChannel(ctx context.Context, mfxChanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_channel %s, took %s to complete", mfxChanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(ctx, mfxChanID)
}

func (lm loggingMiddleware) ConnectThing(ctx context.Context, mfxChanID, mfxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("connect_thing for channel %s and thing %s, took %s to complete", mfxChanID, mfxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConnectThing(ctx, mfxChanID, mfxThingID)
}

func (lm loggingMiddleware) DisconnectThing(ctx context.Context, mfxChanID, mfxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("disconnect_thing mfx-%s : mfx-%s, took %s to complete", mfxChanID, mfxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThing(ctx, mfxChanID, mfxThingID)
}

func (lm loggingMiddleware) Browse(ctx context.Context, serverURI, namespace, identifier string) (nodes []opcua.BrowsedNode, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("browse server URI %s and node %s;%s, took %s to complete", serverURI, namespace, identifier, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Browse(ctx, serverURI, namespace, identifier)
}
