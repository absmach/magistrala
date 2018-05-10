// +build !test

package api

import (
	"fmt"
	"time"

	"github.com/mainflux/mainflux/clients"
	log "github.com/mainflux/mainflux/logger"
)

var _ clients.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    clients.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc clients.Service, logger log.Logger) clients.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) AddClient(key string, client clients.Client) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method add_client for key %s and client %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddClient(key, client)
}

func (lm *loggingMiddleware) UpdateClient(key string, client clients.Client) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_client for key %s and client %s took %s to complete", key, client.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateClient(key, client)
}

func (lm *loggingMiddleware) ViewClient(key string, id string) (client clients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_client for key %s and client %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewClient(key, id)
}

func (lm *loggingMiddleware) ListClients(key string, offset, limit int) (clients []clients.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_clients for key %s took %s to complete", key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListClients(key, offset, limit)
}

func (lm *loggingMiddleware) RemoveClient(key string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_client for key %s and client %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveClient(key, id)
}

func (lm *loggingMiddleware) CreateChannel(key string, channel clients.Channel) (id string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_channel for key %s and channel %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateChannel(key, channel)
}

func (lm *loggingMiddleware) UpdateChannel(key string, channel clients.Channel) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_channel for key %s and channel %s took %s to complete", key, channel.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateChannel(key, channel)
}

func (lm *loggingMiddleware) ViewChannel(key string, id string) (channel clients.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_channel for key %s and channel %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewChannel(key, id)
}

func (lm *loggingMiddleware) ListChannels(key string, offset, limit int) (channels []clients.Channel, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_channels for key %s took %s to complete", key, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListChannels(key, offset, limit)
}

func (lm *loggingMiddleware) RemoveChannel(key string, id string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_channel for key %s and channel %s took %s to complete", key, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveChannel(key, id)
}

func (lm *loggingMiddleware) Connect(key, chanID, clientID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method connect for key %s, channel %s, client %s took %s to complete", key, chanID, clientID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Connect(key, chanID, clientID)
}

func (lm *loggingMiddleware) Disconnect(key, chanID, clientID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method disconnect for key %s, channel %s, client %s took %s to complete", key, chanID, clientID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Disconnect(key, chanID, clientID)
}

func (lm *loggingMiddleware) CanAccess(key string, id string) (pub string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method can_access for key %s, channel %s and publisher %s took %s to complete", key, id, pub, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CanAccess(key, id)
}
