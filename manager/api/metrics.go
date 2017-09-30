package api

import (
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/manager"
)

var _ manager.Service = (*metricService)(nil)

type metricService struct {
	counter metrics.Counter
	latency metrics.Histogram
	manager.Service
}

// NewMetricService instruments core service by tracking request count and
// latency.
func NewMetricService(counter metrics.Counter, latency metrics.Histogram, s manager.Service) manager.Service {
	return &metricService{
		counter: counter,
		latency: latency,
		Service: s,
	}
}

func (ms *metricService) Register(user manager.User) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "register").Add(1)
		ms.latency.With("method", "register").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.Register(user)
}

func (ms *metricService) Login(user manager.User) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "login").Add(1)
		ms.latency.With("method", "login").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.Login(user)
}

func (ms *metricService) Identity(key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identity").Add(1)
		ms.latency.With("method", "identity").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.Identity(key)
}

func (ms *metricService) AddClient(key string, client manager.Client) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "add_client").Add(1)
		ms.latency.With("method", "add_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.AddClient(key, client)
}

func (ms *metricService) UpdateClient(key string, client manager.Client) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_client").Add(1)
		ms.latency.With("method", "update_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.UpdateClient(key, client)
}

func (ms *metricService) ViewClient(key string, id string) (manager.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_client").Add(1)
		ms.latency.With("method", "view_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.ViewClient(key, id)
}

func (ms *metricService) ListClients(key string) ([]manager.Client, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_clients").Add(1)
		ms.latency.With("method", "list_clients").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.ListClients(key)
}

func (ms *metricService) RemoveClient(key string, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_client").Add(1)
		ms.latency.With("method", "remove_client").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.RemoveClient(key, id)
}

func (ms *metricService) CreateChannel(key string, channel manager.Channel) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_channel").Add(1)
		ms.latency.With("method", "create_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.CreateChannel(key, channel)
}

func (ms *metricService) UpdateChannel(key string, channel manager.Channel) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_channel").Add(1)
		ms.latency.With("method", "update_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.UpdateChannel(key, channel)
}

func (ms *metricService) ViewChannel(key string, id string) (manager.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_channel").Add(1)
		ms.latency.With("method", "view_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.ViewChannel(key, id)
}

func (ms *metricService) ListChannels(key string) ([]manager.Channel, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_channels").Add(1)
		ms.latency.With("method", "list_channels").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.ListChannels(key)
}

func (ms *metricService) RemoveChannel(key string, id string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_channel").Add(1)
		ms.latency.With("method", "remove_channel").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.RemoveChannel(key, id)
}

func (ms *metricService) CanAccess(key string, id string) bool {
	defer func(begin time.Time) {
		ms.counter.With("method", "can_access").Add(1)
		ms.latency.With("method", "can_access").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.Service.CanAccess(key, id)
}
