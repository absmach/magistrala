// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
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

func (lm loggingMiddleware) CreateThing(mfxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing %s with NodeID %s, took %s to complete", mfxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(mfxThing, opcuaNodeID)
}

func (lm loggingMiddleware) UpdateThing(mfxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing %s with NodeID %s, took %s to complete", mfxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(mfxThing, opcuaNodeID)
}

func (lm loggingMiddleware) RemoveThing(mfxThing string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing %s, took %s to complete", mfxThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(mfxThing)
}

func (lm loggingMiddleware) CreateChannel(mfxChan, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel %s with ServerURI %s, took %s to complete", mfxChan, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(mfxChan, opcuaServerURI)
}

func (lm loggingMiddleware) UpdateChannel(mfxChanID, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel %s with ServerURI %s, took %s to complete", mfxChanID, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(mfxChanID, opcuaServerURI)
}

func (lm loggingMiddleware) RemoveChannel(mfxChanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_channel %s, took %s to complete", mfxChanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(mfxChanID)
}

func (lm loggingMiddleware) ConnectThing(mfxChanID, mfxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("connect_thing for channel %s and thing %s, took %s to complete", mfxChanID, mfxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConnectThing(mfxChanID, mfxThingID)
}

func (lm loggingMiddleware) DisconnectThing(mfxChanID, mfxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("disconnect_thing mfx-%s : mfx-%s, took %s to complete", mfxChanID, mfxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThing(mfxChanID, mfxThingID)
}

func (lm loggingMiddleware) Browse(serverURI, namespace, identifier string) (nodes []opcua.BrowsedNode, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("browse server URI %s and node %s;%s, took %s to complete", serverURI, namespace, identifier, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Browse(serverURI, namespace, identifier)
}
