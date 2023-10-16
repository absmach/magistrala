package observation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
	"github.com/plgd-dev/go-coap/v3/message/pool"
	"github.com/plgd-dev/go-coap/v3/net/responsewriter"
	"github.com/plgd-dev/go-coap/v3/pkg/errors"
	coapSync "github.com/plgd-dev/go-coap/v3/pkg/sync"
	"go.uber.org/atomic"
)

type DoFunc = func(req *pool.Message) (*pool.Message, error)

type Client interface {
	Context() context.Context
	WriteMessage(req *pool.Message) error
	ReleaseMessage(msg *pool.Message)
	AcquireMessage(ctx context.Context) *pool.Message
}

// The HandlerFunc type is an adapter to allow the use of
// ordinary functions as COAP handlers.
type HandlerFunc[C Client] func(*responsewriter.ResponseWriter[C], *pool.Message)

type Handler[C Client] struct {
	cc           C
	observations *coapSync.Map[uint64, *Observation[C]]
	next         HandlerFunc[C]
	do           DoFunc
}

func (h *Handler[C]) Handle(w *responsewriter.ResponseWriter[C], r *pool.Message) {
	if o, ok := h.observations.Load(r.Token().Hash()); ok {
		o.handle(r)
		return
	}
	h.next(w, r)
}

func (h *Handler[C]) client() C {
	return h.cc
}

func (h *Handler[C]) NewObservation(req *pool.Message, observeFunc func(req *pool.Message)) (*Observation[C], error) {
	observe, err := req.Observe()
	if err != nil {
		return nil, fmt.Errorf("cannot get observe option: %w", err)
	}
	if observe != 0 {
		return nil, fmt.Errorf("invalid value of observe(%v): expected 0", observe)
	}
	token := req.Token()
	if len(token) == 0 {
		return nil, fmt.Errorf("empty token")
	}
	options, err := req.Options().Clone()
	if err != nil {
		return nil, fmt.Errorf("cannot clone options: %w", err)
	}
	respObservationChan := make(chan respObservationMessage, 1)
	o := newObservation(message.Message{
		Token:   req.Token(),
		Code:    req.Code(),
		Options: options,
	}, h, observeFunc, respObservationChan)
	defer func(err *error) {
		if *err != nil {
			o.cleanUp()
		}
	}(&err)
	if _, loaded := h.observations.LoadOrStore(token.Hash(), o); loaded {
		err = errors.ErrKeyAlreadyExists
		return nil, err
	}

	err = h.cc.WriteMessage(req)
	if err != nil {
		return nil, err
	}
	select {
	case <-req.Context().Done():
		err = req.Context().Err()
		return nil, err
	case <-h.cc.Context().Done():
		err = fmt.Errorf("connection was closed: %w", h.cc.Context().Err())
		return nil, err
	case resp := <-respObservationChan:
		if resp.code != codes.Content && resp.code != codes.Valid {
			err = fmt.Errorf("unexpected return code(%v)", resp.code)
			return nil, err
		}
		if resp.notSupported {
			o.cleanUp()
		}
		return o, nil
	}
}

func (h *Handler[C]) GetObservation(key uint64) (*Observation[C], bool) {
	return h.observations.Load(key)
}

// GetObservationRequest returns observation request for token
func (h *Handler[C]) GetObservationRequest(token message.Token) (*pool.Message, bool) {
	obs, ok := h.GetObservation(token.Hash())
	if !ok {
		return nil, false
	}
	req := obs.Request()
	msg := h.cc.AcquireMessage(h.cc.Context())
	msg.ResetOptionsTo(req.Options)
	msg.SetCode(req.Code)
	msg.SetToken(req.Token)
	return msg, true
}

func (h *Handler[C]) pullOutObservation(key uint64) (*Observation[C], bool) {
	return h.observations.LoadAndDelete(key)
}

func NewHandler[C Client](cc C, next HandlerFunc[C], do DoFunc) *Handler[C] {
	return &Handler[C]{
		cc:           cc,
		observations: coapSync.NewMap[uint64, *Observation[C]](),
		next:         next,
		do:           do,
	}
}

type respObservationMessage struct {
	code         codes.Code
	notSupported bool
}

// Observation represents subscription to resource on the server
type Observation[C Client] struct {
	req                 message.Message
	observeFunc         func(req *pool.Message)
	respObservationChan chan respObservationMessage
	waitForResponse     atomic.Bool
	observationHandler  *Handler[C]

	private struct { // members guarded by mutex
		mutex       sync.Mutex
		obsSequence uint32
		lastEvent   time.Time
		etag        []byte
	}
}

func (o *Observation[C]) Canceled() bool {
	_, ok := o.observationHandler.GetObservation(o.req.Token.Hash())
	return !ok
}

func newObservation[C Client](req message.Message, observationHandler *Handler[C], observeFunc func(req *pool.Message), respObservationChan chan respObservationMessage) *Observation[C] {
	return &Observation[C]{
		req:                 req,
		waitForResponse:     *atomic.NewBool(true),
		respObservationChan: respObservationChan,
		observeFunc:         observeFunc,
		observationHandler:  observationHandler,
	}
}

func (o *Observation[C]) handle(r *pool.Message) {
	if o.waitForResponse.CompareAndSwap(true, false) {
		select {
		case o.respObservationChan <- respObservationMessage{
			code:         r.Code(),
			notSupported: !r.HasOption(message.Observe),
		}:
		default:
		}
		o.respObservationChan = nil
	}
	if o.wantBeNotified(r) {
		o.observeFunc(r)
	}
}

func (o *Observation[C]) cleanUp() bool {
	// we can ignore err during cleanUp, if err != nil then some other
	// part of code already removed the handler for the token
	_, ok := o.observationHandler.pullOutObservation(o.req.Token.Hash())
	return ok
}

func (o *Observation[C]) client() C {
	return o.observationHandler.client()
}

func (o *Observation[C]) Request() message.Message {
	return o.req
}

func (o *Observation[C]) etag() []byte {
	o.private.mutex.Lock()
	defer o.private.mutex.Unlock()
	return o.private.etag
}

// Cancel remove observation from server. For recreate observation use Observe.
func (o *Observation[C]) Cancel(ctx context.Context, opts ...message.Option) error {
	if !o.cleanUp() {
		// observation was already cleanup
		return nil
	}

	req := o.client().AcquireMessage(ctx)
	defer o.client().ReleaseMessage(req)
	req.ResetOptionsTo(opts)
	req.SetCode(codes.GET)
	req.SetObserve(1)
	if path, err := o.req.Options.Path(); err == nil {
		if err := req.SetPath(path); err != nil {
			return fmt.Errorf("cannot set path(%v): %w", path, err)
		}
	}
	req.SetToken(o.req.Token)
	etag := o.etag()
	if len(etag) > 0 {
		_ = req.SetETag(etag) // ignore invalid etag
	}
	resp, err := o.observationHandler.do(req)
	if err != nil {
		return err
	}
	defer o.client().ReleaseMessage(resp)
	if resp.Code() != codes.Content && resp.Code() != codes.Valid {
		return fmt.Errorf("unexpected return code(%v)", resp.Code())
	}
	return nil
}

func (o *Observation[C]) wantBeNotified(r *pool.Message) bool {
	obsSequence, err := r.Observe()
	if err != nil {
		return true
	}
	now := time.Now()

	o.private.mutex.Lock()
	defer o.private.mutex.Unlock()
	if !ValidSequenceNumber(o.private.obsSequence, obsSequence, o.private.lastEvent, now) {
		return false
	}

	o.private.obsSequence = obsSequence
	o.private.lastEvent = now
	if etag, err := r.ETag(); err == nil {
		if cap(o.private.etag) < len(etag) {
			o.private.etag = make([]byte, len(etag))
		}
		if len(o.private.etag) != len(etag) {
			o.private.etag = o.private.etag[:len(etag)]
		}
		copy(o.private.etag, etag)
	}
	return true
}
