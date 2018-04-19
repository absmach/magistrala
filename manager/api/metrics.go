// +build !test

package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     manager.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc manager.Service, counter metrics.Counter, latency metrics.Histogram) manager.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Register(user manager.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "register").Add(1)
		ms.latency.With("method", "register").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Register(user)
}

func (ms *metricsMiddleware) Login(user manager.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "login").Add(1)
		ms.latency.With("method", "login").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Login(user)
}

func (ms *metricsMiddleware) AddClient(key string, client manager.Client) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_client").Add(1)
		ms.latency.With("method", "add_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AddClient(key, client)
}

func (ms *metricsMiddleware) UpdateClient(key string, client manager.Client) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client").Add(1)
		ms.latency.With("method", "update_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateClient(key, client)
}

func (ms *metricsMiddleware) ViewClient(key string, id string) (manager.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewClient(key, id)
}

func (ms *metricsMiddleware) ListClients(key string, offset, limit int) ([]manager.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListClients(key, offset, limit)
}

func (ms *metricsMiddleware) RemoveClient(key string, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_client").Add(1)
		ms.latency.With("method", "remove_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveClient(key, id)
}

func (ms *metricsMiddleware) CreateChannel(key string, channel manager.Channel) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_channel").Add(1)
		ms.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateChannel(key, channel)
}

func (ms *metricsMiddleware) UpdateChannel(key string, channel manager.Channel) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateChannel(key, channel)
}

func (ms *metricsMiddleware) ViewChannel(key string, id string) (manager.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewChannel(key, id)
}

func (ms *metricsMiddleware) ListChannels(key string, offset, limit int) ([]manager.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListChannels(key, offset, limit)
}

func (ms *metricsMiddleware) RemoveChannel(key string, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_channel").Add(1)
		ms.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveChannel(key, id)
}

func (ms *metricsMiddleware) Connect(key, chanId, clientId string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Connect(key, chanId, clientId)
}

func (ms *metricsMiddleware) Disconnect(key, chanId, clientId string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Disconnect(key, chanId, clientId)
}

func (ms *metricsMiddleware) Identity(key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identity").Add(1)
		ms.latency.With("method", "identity").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identity(key)
}

func (ms *metricsMiddleware) CanAccess(key string, id string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "can_access").Add(1)
		ms.latency.With("method", "can_access").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CanAccess(key, id)
}
