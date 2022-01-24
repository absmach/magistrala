// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"fmt"
	"time"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/provision"
)

var _ provision.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    provision.Service
}

// NewLoggingMiddleware adds logging facilities to the core service.
func NewLoggingMiddleware(svc provision.Service, logger log.Logger) provision.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Provision(token, name, externalID, externalKey string) (res provision.Result, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method provision for token: %s and things: %v took %s to complete", token, res.Things, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors", message))
	}(time.Now())

	return lm.svc.Provision(token, name, externalID, externalKey)
}

func (lm *loggingMiddleware) Cert(token, thingID, duration string, keyBits int) (cert string, key string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method cert for token: %s and thing: %v took %s to complete", token, thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors", message))
	}(time.Now())

	return lm.svc.Cert(token, thingID, duration, keyBits)
}

func (lm *loggingMiddleware) Mapping(token string) (res map[string]interface{}, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method mapping for token: %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors", message))
	}(time.Now())

	return lm.svc.Mapping(token)
}
