// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	mglog "github.com/absmach/magistrala/logger"
	"github.com/absmach/magistrala/opcua"
)

var _ opcua.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger mglog.Logger
	svc    opcua.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc opcua.Service, logger mglog.Logger) opcua.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm loggingMiddleware) CreateThing(ctx context.Context, mgxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_thing %s with NodeID %s, took %s to complete", mgxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateThing(ctx, mgxThing, opcuaNodeID)
}

func (lm loggingMiddleware) UpdateThing(ctx context.Context, mgxThing, opcuaNodeID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_thing %s with NodeID %s, took %s to complete", mgxThing, opcuaNodeID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateThing(ctx, mgxThing, opcuaNodeID)
}

func (lm loggingMiddleware) RemoveThing(ctx context.Context, mgxThing string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_thing %s, took %s to complete", mgxThing, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveThing(ctx, mgxThing)
}

func (lm loggingMiddleware) CreateChannel(ctx context.Context, mgxChan, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("create_channel %s with ServerURI %s, took %s to complete", mgxChan, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(ctx, mgxChan, opcuaServerURI)
}

func (lm loggingMiddleware) UpdateChannel(ctx context.Context, mgxChanID, opcuaServerURI string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("update_channel %s with ServerURI %s, took %s to complete", mgxChanID, opcuaServerURI, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(ctx, mgxChanID, opcuaServerURI)
}

func (lm loggingMiddleware) RemoveChannel(ctx context.Context, mgxChanID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("remove_channel %s, took %s to complete", mgxChanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(ctx, mgxChanID)
}

func (lm loggingMiddleware) ConnectThing(ctx context.Context, mgxChanID, mgxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("connect_thing for channel %s and thing %s, took %s to complete", mgxChanID, mgxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConnectThing(ctx, mgxChanID, mgxThingID)
}

func (lm loggingMiddleware) DisconnectThing(ctx context.Context, mgxChanID, mgxThingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("disconnect_thing mgx-%s : mgx-%s, took %s to complete", mgxChanID, mgxThingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DisconnectThing(ctx, mgxChanID, mgxThingID)
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
