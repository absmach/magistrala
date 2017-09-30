package api

import (
	"time"

	"github.com/go-kit/kit/log"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.Service = (*loggingService)(nil)

type loggingService struct {
	logger log.Logger
	manager.Service
}

// NewLoggingService adds logging facilities to the core service.
func NewLoggingService(logger log.Logger, s manager.Service) manager.Service {
	return &loggingService{logger, s}
}

func (ls *loggingService) Register(user manager.User) (err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "register",
			"email", user.Email,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.Register(user)
}

func (ls *loggingService) Login(user manager.User) (token string, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "login",
			"email", user.Email,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.Login(user)
}

func (ls *loggingService) Identity(key string) (id string, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "identity",
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.Identity(key)
}

func (ls *loggingService) AddClient(key string, client manager.Client) (id string, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "add_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.AddClient(key, client)
}

func (ls *loggingService) UpdateClient(key string, client manager.Client) (err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "update_client",
			"key", key,
			"id", client.ID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.UpdateClient(key, client)
}

func (ls *loggingService) ViewClient(key string, id string) (client manager.Client, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "view_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.ViewClient(key, id)
}

func (ls *loggingService) ListClients(key string) (clients []manager.Client, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "list_clients",
			"key", key,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.ListClients(key)
}

func (ls *loggingService) RemoveClient(key string, id string) (err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "remove_client",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.RemoveClient(key, id)
}

func (ls *loggingService) CreateChannel(key string, channel manager.Channel) (id string, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "create_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.CreateChannel(key, channel)
}

func (ls *loggingService) UpdateChannel(key string, channel manager.Channel) (err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "update_channel",
			"key", key,
			"id", channel.ID,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.UpdateChannel(key, channel)
}

func (ls *loggingService) ViewChannel(key string, id string) (channel manager.Channel, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "view_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.ViewChannel(key, id)
}

func (ls *loggingService) ListChannels(key string) (channels []manager.Channel, err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "list_channels",
			"key", key,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.ListChannels(key)
}

func (ls *loggingService) RemoveChannel(key string, id string) (err error) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "remove_channel",
			"key", key,
			"id", id,
			"error", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.RemoveChannel(key, id)
}

func (ls *loggingService) CanAccess(key string, id string) (allowed bool) {
	defer func(begin time.Time) {
		ls.logger.Log(
			"method", "can_access",
			"key", key,
			"id", id,
			"allowed", allowed,
			"took", time.Since(begin),
		)
	}(time.Now())

	return ls.Service.CanAccess(key, id)
}
