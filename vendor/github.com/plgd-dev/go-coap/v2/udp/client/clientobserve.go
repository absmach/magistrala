package client

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/udp/message/pool"
)

//Observation represents subscription to resource on the server
type Observation struct {
	token       message.Token
	path        string
	obsSequence uint32
	etag        []byte
	cc          *ClientConn

	mutex sync.Mutex
}

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

func (o *Observation) cleanUp() {
	o.cc.observationTokenHandler.Pop(o.token)
	registeredRequest, ok := o.cc.observationRequests.PullOut(o.token.String())
	if ok {
		pool.ReleaseMessage(registeredRequest.(*pool.Message))
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
	return o.cc.WriteMessage(req)
}

func (o *Observation) wantBeNotified(r *pool.Message) bool {
	obsSequence, err := r.Observe()
	if err != nil {
		return true
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()
	//obs starts with 0, after that check obsSequence
	if obsSequence != 0 && o.obsSequence > obsSequence {
		return false
	}
	o.obsSequence = obsSequence

	return true
}

// Observe subscribes for every change of resource on path.
func (cc *ClientConn) Observe(ctx context.Context, path string, observeFunc func(req *pool.Message), opts ...message.Option) (*Observation, error) {
	req, err := NewGetRequest(ctx, path, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create observe request: %w", err)
	}
	token := req.Token()
	req.SetObserve(0)
	o := &Observation{
		token:       token,
		path:        path,
		obsSequence: 0,
		cc:          cc,
	}
	respCodeChan := make(chan codes.Code, 1)
	waitForReponse := uint32(1)
	cc.observationRequests.Store(token.String(), req)
	err = o.cc.observationTokenHandler.Insert(token.String(), func(w *ResponseWriter, r *pool.Message) {
		code := r.Code()
		if atomic.CompareAndSwapUint32(&waitForReponse, 1, 0) {
			select {
			case respCodeChan <- code:
			default:
			}
		}
		if o.wantBeNotified(r) {
			observeFunc(r)
		}
	})
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
			err = fmt.Errorf("unexected return code(%v)", respCode)
			return nil, err
		}
		return o, nil
	}
}
