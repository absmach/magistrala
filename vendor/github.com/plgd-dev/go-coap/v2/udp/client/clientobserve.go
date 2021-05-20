package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/net/observation"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

func NewObservationHandler(obsertionTokenHandler *HandlerContainer, next HandlerFunc) HandlerFunc {
	return func(w *ResponseWriter, r *pool.Message) {
		v, err := obsertionTokenHandler.Get(r.Token())
		if err == nil {
			v(w, r)
			return
		}
		obs, err := r.Observe()
		if err == nil && obs > 1 {
			w.SendReset()
			return
		}
		next(w, r)
	}
}

//Observation represents subscription to resource on the server
type Observation struct {
	token        message.Token
	path         string
	cc           *ClientConn
	observeFunc  func(req *pool.Message)
	respCodeChan chan codes.Code

	obsSequence uint32
	etag        []byte
	lastEvent   time.Time
	mutex       sync.Mutex

	waitForReponse uint32
}

func newObservation(token message.Token, path string, cc *ClientConn, observeFunc func(req *pool.Message), respCodeChan chan codes.Code) *Observation {
	return &Observation{
		token:          token,
		path:           path,
		obsSequence:    0,
		cc:             cc,
		waitForReponse: 1,
		respCodeChan:   respCodeChan,
		observeFunc:    observeFunc,
	}
}

func (o *Observation) cleanUp() {
	o.cc.observationTokenHandler.Pop(o.token)
	registeredRequest, ok := o.cc.observationRequests.PullOut(o.token.String())
	if ok {
		pool.ReleaseMessage(registeredRequest.(*pool.Message))
	}
}

func (o *Observation) handler(w *ResponseWriter, r *pool.Message) {
	code := r.Code()
	if atomic.CompareAndSwapUint32(&o.waitForReponse, 1, 0) {
		select {
		case o.respCodeChan <- code:
		default:
		}
		o.respCodeChan = nil
	}
	if o.wantBeNotified(r) {
		o.observeFunc(r)
	}
}

// Cancel remove observation from server. For recreate observation use Observe.
func (o *Observation) Cancel(ctx context.Context) error {
	o.cleanUp()
	req, err := NewGetRequest(ctx, o.path)
	if err != nil {
		return fmt.Errorf("cannot cancel observation request: %w", err)
	}
	defer pool.ReleaseMessage(req)
	req.SetObserve(1)
	req.SetToken(o.token)
	resp, err := o.cc.Do(req)
	if err != nil {
		return err
	}
	defer pool.ReleaseMessage(resp)
	if resp.Code() != codes.Content {
		return fmt.Errorf("unexpected return code(%v)", resp.Code())
	}
	return err
}

func (o *Observation) wantBeNotified(r *pool.Message) bool {
	obsSequence, err := r.Observe()
	if err != nil {
		return true
	}
	now := time.Now()

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if observation.ValidSequenceNumber(o.obsSequence, obsSequence, o.lastEvent, now) {
		o.obsSequence = obsSequence
		o.lastEvent = now
		return true
	}

	return false
}

// Observe subscribes for every change of resource on path.
func (cc *ClientConn) Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (*Observation, error) {
	req, err := NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create observe request: %w", err)
	}
	token := req.Token()
	req.SetObserve(0)
	respCodeChan := make(chan codes.Code, 1)
	o := newObservation(token, path, cc, observeFunc, respCodeChan)

	cc.observationRequests.Store(token.String(), req)
	err = o.cc.observationTokenHandler.Insert(token.String(), o.handler)
	defer func(err *error) {
		if *err != nil {
			o.cleanUp()
		}
	}(&err)
	if err != nil {
		return nil, err
	}

	err = cc.WriteMessage(req)
	if err != nil {
		return nil, err
	}
	select {
	case <-req.Context().Done():
		err = req.Context().Err()
		return nil, err
	case <-cc.Context().Done():
		err = fmt.Errorf("connection was closed: %w", cc.Context().Err())
		return nil, err
	case respCode := <-respCodeChan:
		if respCode != codes.Content {
			err = fmt.Errorf("unexpected return code(%v)", respCode)
			return nil, err
		}
		return o, nil
	}
}
