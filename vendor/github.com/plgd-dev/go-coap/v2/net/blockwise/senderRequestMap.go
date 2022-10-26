package blockwise

import (
	"fmt"

	kitSync "github.com/plgd-dev/kit/v2/sync"
)

func messageToGuardTransferKey(msg Message) string {
	code := msg.Code()
	path, _ := msg.Path()
	queries, _ := msg.Queries()

	return fmt.Sprintf("%v:%v:%v", code, path, queries)
}

type senderRequest struct {
	transferKey string
	*messageGuard
	release func()
	lock    bool
}

func setTypeFrom(to Message, from Message) {
	if udpTo, ok := to.(hasType); ok {
		if udpFrom, ok := from.(hasType); ok {
			udpTo.SetType(udpFrom.Type())
		}
	}
}

func (b *BlockWise) newSentRequestMessage(r Message, lock bool) *senderRequest {
	req := b.acquireMessage(r.Context())
	req.SetCode(r.Code())
	req.SetToken(r.Token())
	req.ResetOptionsTo(r.Options())
	setTypeFrom(req, r)
	data := &senderRequest{
		transferKey:  messageToGuardTransferKey(req),
		messageGuard: newRequestGuard(req),
		release: func() {
			b.releaseMessage(req)
		},
		lock: lock,
	}
	return data
}

type senderRequestMap struct {
	byToken       *kitSync.Map
	byTransferKey *kitSync.Map
}

func newSenderRequestMap() *senderRequestMap {
	return &senderRequestMap{
		byToken:       kitSync.NewMap(),
		byTransferKey: kitSync.NewMap(),
	}
}

func (m *senderRequestMap) store(req *senderRequest) error {
	if !req.lock {
		m.byToken.Store(req.Token().Hash(), req)
		return nil
	}
	for {
		var err error
		v, loaded := m.byTransferKey.LoadOrStoreWithFunc(req.transferKey, func(value interface{}) interface{} {
			return value
		}, func() interface{} {
			err = req.Acquire(req.Context(), 1)
			return req
		})
		if err != nil {
			return fmt.Errorf("cannot lock message: %w", err)
		}
		if !loaded {
			m.byToken.Store(req.Token().Hash(), req)
			return nil
		}
		p := v.(*senderRequest)
		err = p.Acquire(req.Context(), 1)
		if err != nil {
			return fmt.Errorf("cannot lock message: %w", err)
		}
		p.Release(1)
	}
}

func (m *senderRequestMap) loadByTokenWithFunc(token uint64, onload func(value *senderRequest) interface{}) interface{} {
	v, ok := m.byToken.LoadWithFunc(token, func(value interface{}) interface{} {
		v := value.(*senderRequest)
		return onload(v)
	})
	if ok {
		return v
	}
	return nil
}

func (m *senderRequestMap) deleteByToken(token uint64) {
	v, ok := m.byToken.PullOut(token)
	if !ok {
		return
	}
	req := v.(*senderRequest)
	v1, ok := m.byTransferKey.Load(req.transferKey)
	if !ok {
		return
	}
	req1 := v1.(*senderRequest)
	if req == req1 {
		m.byTransferKey.Delete(req.transferKey)
		req.Release(1)
	}
}
