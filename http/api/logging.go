package api

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/mainflux/mainflux/http"
	"github.com/mainflux/mainflux/writer"
)

var _ http.Service = (*loggingService)(nil)

type loggingService struct {
	logger log.Logger
	http.Service
}

// NewLoggingService adds logging facilities to the adapter.
func NewLoggingService(logger log.Logger, s http.Service) http.Service {
	return &loggingService{logger, s}
}

func (ls *loggingService) Publish(msg writer.RawMessage) error {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "publish",
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.Publish(msg)
}
