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

var (
	errEntryNotFound = errors.New("observer entry not founds")
)

const (
	// Default service transmission settings from RFC.
	// It could be changed depending on specific use-case.

	// MaxRetransmit is the maximum number of times a message will be retransmitted.
	MaxRetransmit = 4
	// AckRandomFactor is a multiplier for response backoff.
	AckRandomFactor = 1.5
	// AckTimeout is the amount of time to wait for a response.
	AckTimeout = int(2 * time.Second)
)

// MaxTimeout is extracted to into a separate variable so there is no
// need for it to be calculated over again.
var MaxTimeout = int(float64(AckTimeout) * ((math.Pow(2, float64(MaxRetransmit))) - 1) * AckRandomFactor)

// AdapterService struct represents CoAP adapter service implementation.
type adapterService struct {
	pubsub nats.Service
	subs   map[string]nats.Channel
	mu     sync.Mutex
}

// New creates new CoAP adapter service struct.
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
		return nil, errEntryNotFound
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
				if duration >= MaxTimeout {
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
