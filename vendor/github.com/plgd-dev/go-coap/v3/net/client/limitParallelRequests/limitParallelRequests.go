package limitparallelrequests

import (
	"context"
	"fmt"
	"hash/crc64"
	"math"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"golang.org/x/sync/semaphore"
)

type (
	DoFunc        = func(req *pool.Message) (*pool.Message, error)
	DoObserveFunc = func(req *pool.Message, observeFunc func(req *pool.Message)) (Observation, error)
)

type Observation = interface {
	Cancel(ctx context.Context, opts ...message.Option) error
	Canceled() bool
}

type endpointQueue struct {
	processedCounter int64
	orderedRequest   []chan struct{}
}

type LimitParallelRequests struct {
	endpointLimit int64
	limit         *semaphore.Weighted
	do            DoFunc
	doObserve     DoObserveFunc
	// only one request can be processed by one endpoint
	endpointQueues *coapSync.Map[uint64, *endpointQueue]
}

// New creates new LimitParallelRequests. When limit, endpointLimit == 0, then limit is not used.
func New(limit, endpointLimit int64, do DoFunc, doObserve DoObserveFunc) *LimitParallelRequests {
	if limit <= 0 {
		limit = math.MaxInt64
	}
	if endpointLimit <= 0 {
		endpointLimit = math.MaxInt64
	}
	return &LimitParallelRequests{
		limit:          semaphore.NewWeighted(limit),
		endpointLimit:  endpointLimit,
		do:             do,
		doObserve:      doObserve,
		endpointQueues: coapSync.NewMap[uint64, *endpointQueue](),
	}
}

func hash(opts message.Options) uint64 {
	h := crc64.New(crc64.MakeTable(crc64.ISO))
	for _, opt := range opts {
		if opt.ID == message.URIPath {
			_, _ = h.Write(opt.Value) // hash never returns an error
		}
	}
	return h.Sum64()
}

func (c *LimitParallelRequests) acquireEndpoint(ctx context.Context, endpointLimitKey uint64) error {
	reqChan := make(chan struct{}) // channel is closed when request can be processed by releaseEndpoint
	_, _ = c.endpointQueues.LoadOrStoreWithFunc(endpointLimitKey, func(value *endpointQueue) *endpointQueue {
		if value.processedCounter < c.endpointLimit {
			close(reqChan)
			value.processedCounter++
			return value
		}
		value.orderedRequest = append(value.orderedRequest, reqChan)
		return value
	}, func() *endpointQueue {
		close(reqChan)
		return &endpointQueue{
			processedCounter: 1,
		}
	})
	select {
	case <-ctx.Done():
		c.releaseEndpoint(endpointLimitKey)
		return ctx.Err()
	case <-reqChan:
		return nil
	}
}

func (c *LimitParallelRequests) releaseEndpoint(endpointLimitKey uint64) {
	_, _ = c.endpointQueues.ReplaceWithFunc(endpointLimitKey, func(oldValue *endpointQueue, oldLoaded bool) (newValue *endpointQueue, doDelete bool) {
		if oldLoaded {
			if len(oldValue.orderedRequest) > 0 {
				reqChan := oldValue.orderedRequest[0]
				oldValue.orderedRequest = oldValue.orderedRequest[1:]
				close(reqChan)
			} else {
				oldValue.processedCounter--
				if oldValue.processedCounter == 0 {
					return nil, true
				}
			}
			return oldValue, false
		}
		return nil, true
	})
}

func (c *LimitParallelRequests) Do(req *pool.Message) (*pool.Message, error) {
	endpointLimitKey := hash(req.Options())
	if err := c.acquireEndpoint(req.Context(), endpointLimitKey); err != nil {
		return nil, fmt.Errorf("cannot process request %v for client endpoint limit: %w", req, err)
	}
	defer c.releaseEndpoint(endpointLimitKey)
	if err := c.limit.Acquire(req.Context(), 1); err != nil {
		return nil, fmt.Errorf("cannot process request %v for client limit: %w", req, err)
	}
	defer c.limit.Release(1)
	return c.do(req)
}

func (c *LimitParallelRequests) DoObserve(req *pool.Message, observeFunc func(req *pool.Message)) (Observation, error) {
	endpointLimitKey := hash(req.Options())
	if err := c.acquireEndpoint(req.Context(), endpointLimitKey); err != nil {
		return nil, fmt.Errorf("cannot process observe request %v for client endpoint limit: %w", req, err)
	}
	defer c.releaseEndpoint(endpointLimitKey)
	err := c.limit.Acquire(req.Context(), 1)
	if err != nil {
		return nil, fmt.Errorf("cannot process observe request %v for client limit: %w", req, err)
	}
	defer c.limit.Release(1)
	return c.doObserve(req, observeFunc)
}
