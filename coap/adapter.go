// Package coap contains the domain concept definitions needed to support
// Mainflux coap adapter service functionality. All constant values are taken
// from RFC, and could be adjusted based on specific use case.
package coap

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap/nats"
	broker "github.com/nats-io/go-nats"
)

const (
	responseBackoffMultiplier = 1.5

	// AckTimeout is the amount of time to wait for a response.
	AckTimeout = int(2 * time.Second)

	// MaxRetransmit is the maximum number of times a message will be retransmitted.
	MaxRetransmit = 4
)

var (
	// ErrFailedMessagePublish indicates that message publishing failed.
	ErrFailedMessagePublish = errors.New("failed to publish message")

	// ErrFailedSubscription indicates that client couldn't subscribe to specified channel.
	ErrFailedSubscription = errors.New("failed to subscribe to a channel")

	// ErrFailedConnection indicates that service couldn't connect to message broker.
	ErrFailedConnection = errors.New("failed to connect to message broker")

	// extracted to avoid recomputation
	maxTimeout = int(float64(AckTimeout) * ((math.Pow(2, float64(MaxRetransmit))) - 1) * responseBackoffMultiplier)
)

// Service specifies coap service API.
type Service interface {
	mainflux.MessagePublisher

	// Subscribes to channel with specified id and adds subscription to
	// service map of subscriptions under given ID.
	Subscribe(string, string, nats.Channel) error

	// Unsubscribe method is used to stop observing resource.
	Unsubscribe(string)

	// SetTimeout sets timeout to wait CONF messages.
	SetTimeout(string, *time.Timer, int) (chan bool, error)

	// RemoveTimeout removes timeout when ACK message is received from client
	// if timeout existed.
	RemoveTimeout(string)
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	pubsub nats.Service
	subs   map[string]nats.Channel
	mu     sync.Mutex
}

// New instantiates the CoAP adapter implementation.
func New(pubsub nats.Service) Service {
	return &adapterService{
		pubsub: pubsub,
		subs:   make(map[string]nats.Channel),
		mu:     sync.Mutex{},
	}
}

func (svc *adapterService) get(clientID string) (nats.Channel, bool) {
	svc.mu.Lock()
	obs, ok := svc.subs[clientID]
	svc.mu.Unlock()
	return obs, ok
}

func (svc *adapterService) put(clientID string, obs nats.Channel) {
	svc.mu.Lock()
	svc.subs[clientID] = obs
	svc.mu.Unlock()
}

func (svc *adapterService) remove(clientID string) {
	svc.mu.Lock()
	obs, ok := svc.subs[clientID]
	if ok {
		obs.Closed <- true
		delete(svc.subs, clientID)
	}
	svc.mu.Unlock()
}

func (svc *adapterService) Publish(msg mainflux.RawMessage) error {
	if err := svc.pubsub.Publish(msg); err != nil {
		switch err {
		case broker.ErrConnectionClosed, broker.ErrInvalidConnection:
			return ErrFailedConnection
		default:
			return ErrFailedMessagePublish
		}
	}
	return nil
}

func (svc *adapterService) Subscribe(chanID, clientID string, ch nats.Channel) error {
	// Remove entry if already exists.
	svc.Unsubscribe(clientID)
	if err := svc.pubsub.Subscribe(chanID, ch); err != nil {
		return ErrFailedSubscription
	}
	svc.put(clientID, ch)
	return nil
}

func (svc *adapterService) Unsubscribe(clientID string) {
	svc.remove(clientID)
}

func (svc *adapterService) SetTimeout(clientID string, timer *time.Timer, duration int) (chan bool, error) {
	sub, ok := svc.get(clientID)
	if !ok {
		return nil, errors.New("observer entry not found")
	}
	go func() {
		for {
			select {
			case _, ok := <-sub.Timer:
				timer.Stop()
				if ok {
					sub.Notify <- false
				}
				return
			case <-timer.C:
				duration *= 2
				if duration >= maxTimeout {
					timer.Stop()
					sub.Notify <- false
					svc.Unsubscribe(clientID)
					return
				}
				timer.Reset(time.Duration(duration))
				sub.Notify <- true
			}
		}
	}()
	return sub.Notify, nil
}

func (svc *adapterService) RemoveTimeout(clientID string) {
	if sub, ok := svc.get(clientID); ok {
		sub.Timer <- false
	}
}
