// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/nats"
	log "github.com/mainflux/mainflux/logger"
)

var _ coap.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    coap.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc coap.Service, logger log.Logger) coap.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(msg mainflux.RawMessage) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish to channel %s took %s to complete", msg.Channel, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(msg)
}

func (lm *loggingMiddleware) Subscribe(chanID, clientID string, channel nats.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method subscribe to channel %s took %s to complete", chanID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(chanID, clientID, channel)
}

func (lm *loggingMiddleware) SetTimeout(clientID string, timer *time.Timer, duration int) (chan bool, error) {
	return lm.svc.SetTimeout(clientID, timer, duration)
}

func (lm *loggingMiddleware) RemoveTimeout(clientID string) {
	lm.svc.RemoveTimeout(clientID)
}

func (lm *loggingMiddleware) Unsubscribe(clientID string) {
	defer func(begin time.Time) {
		lm.logger.Info(fmt.Sprintf("Method unsubscribe for client %s took %s to complete", clientID, time.Since(begin)))
	}(time.Now())
	lm.svc.Unsubscribe(clientID)
}
