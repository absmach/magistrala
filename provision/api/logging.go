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

func (lm *loggingMiddleware) Provision(externalID, externalKey string) (res provision.Result, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add thing and connect to proxy: new thing ID: %s took %s to complete", res.Thing.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Provision(externalID, externalKey)
}
