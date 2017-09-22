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

func (ls *loggingService) Send(msgs []writer.Message) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "send",
			"took", time.Since(begin),
		)
	}(time.Now())

	ls.Service.Send(msgs)
}
