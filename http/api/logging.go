package api

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/mainflux/mainflux"
)

var _ mainflux.MessagePublisher = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    mainflux.MessagePublisher
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc mainflux.MessagePublisher, logger log.Logger) mainflux.MessagePublisher {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(msg mainflux.RawMessage) error {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "publish",
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Publish(msg)
}
