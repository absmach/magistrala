package api

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    manager.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc manager.Service, logger log.Logger) manager.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Register(user manager.User) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "register",
			"email", user.Email,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Register(user)
}

func (lm *loggingMiddleware) Login(user manager.User) (token string, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "login",
			"email", user.Email,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Login(user)
}

func (lm *loggingMiddleware) AddClient(key string, client manager.Client) (id string, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "add_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.AddClient(key, client)
}

func (lm *loggingMiddleware) UpdateClient(key string, client manager.Client) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "update_client",
			"key", key,
			"id", client.ID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.UpdateClient(key, client)
}

func (lm *loggingMiddleware) ViewClient(key string, id string) (client manager.Client, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "view_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.ViewClient(key, id)
}

func (lm *loggingMiddleware) ListClients(key string) (clients []manager.Client, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "list_clients",
			"key", key,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.ListClients(key)
}

func (lm *loggingMiddleware) RemoveClient(key string, id string) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "remove_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.RemoveClient(key, id)
}

func (lm *loggingMiddleware) CreateChannel(key string, channel manager.Channel) (id string, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "create_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.CreateChannel(key, channel)
}

func (lm *loggingMiddleware) UpdateChannel(key string, channel manager.Channel) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "update_channel",
			"key", key,
			"id", channel.ID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.UpdateChannel(key, channel)
}

func (lm *loggingMiddleware) ViewChannel(key string, id string) (channel manager.Channel, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "view_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.ViewChannel(key, id)
}

func (lm *loggingMiddleware) ListChannels(key string) (channels []manager.Channel, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "list_channels",
			"key", key,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.ListChannels(key)
}

func (lm *loggingMiddleware) RemoveChannel(key string, id string) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "remove_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.RemoveChannel(key, id)
}

func (lm *loggingMiddleware) Connect(key, chanId, clientId string) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "connect",
			"key", key,
			"channel", chanId,
			"client", clientId,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Connect(key, chanId, clientId)
}

func (lm *loggingMiddleware) Disconnect(key, chanId, clientId string) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "disconnect",
			"key", key,
			"channel", chanId,
			"client", clientId,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Disconnect(key, chanId, clientId)
}

func (lm *loggingMiddleware) Identity(key string) (id string, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "identity",
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.Identity(key)
}

func (lm *loggingMiddleware) CanAccess(key string, id string) (pub string, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "can_access",
			"key", key,
			"id", id,
			"publisher", pub,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return lm.svc.CanAccess(key, id)
}
