package pool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/message/pool"
	udp "github.com/plgd-dev/go-coap/v2/udp/message"
)

type Message struct {

	//local vars
	rawData        []byte
	rawMarshalData []byte

	ctx       context.Context
	messageID *uint16
	*pool.Message
	rawMaxSize uint16

	typ udp.Type

	isModified bool
}

// Reset clear message for next reuse
func (r *Message) Reset() {
	r.Message.Reset()
	r.messageID = nil
	r.typ = udp.NonConfirmable
	if cap(r.rawData) > int(r.rawMaxSize) {
		r.rawData = make([]byte, 256)
	}
	if cap(r.rawMarshalData) > int(r.rawMaxSize) {
		r.rawMarshalData = make([]byte, 256)
	}
	r.isModified = false
}

func (r *Message) Context() context.Context {
	return r.ctx
}

func (r *Message) SetMessageID(mid uint16) {
	r.messageID = &mid
	r.isModified = true
}

func (r *Message) UpsertMessageID(mid uint16) uint16 {
	if r.messageID != nil {
		return *r.messageID
	}
	r.messageID = &mid
	return mid
}

func (r *Message) MessageID() uint16 {
	if r.messageID == nil {
		panic("messageID is not set")
	}
	return *r.messageID
}

func (r *Message) SetType(typ udp.Type) {
	r.typ = typ
	r.isModified = true
}

func (r *Message) Type() udp.Type {
	return r.typ
}

func (r *Message) IsModified() bool {
	return r.isModified || r.Message.IsModified()
}

func (r *Message) SetModified(b bool) {
	r.isModified = b
	r.Message.SetModified(b)
}

func (r *Message) Unmarshal(data []byte) (int, error) {
	if len(r.rawData) < len(data) {
		r.rawData = append(r.rawData, make([]byte, len(data)-len(r.rawData))...)
	}
	copy(r.rawData, data)
	r.rawData = r.rawData[:len(data)]
	m := &udp.Message{
		Options: make(message.Options, 0, 16),
	}

	n, err := m.Unmarshal(r.rawData)
	if err != nil {
		return n, err
	}
	r.Message.SetCode(m.Code)
	r.Message.SetToken(m.Token)
	r.Message.ResetOptionsTo(m.Options)
	r.typ = m.Type
	r.messageID = &m.MessageID
	if len(m.Payload) > 0 {
		r.Message.SetBody(bytes.NewReader(m.Payload))
	}
	return n, err
}

func (r *Message) Marshal() ([]byte, error) {
	m := udp.Message{
		Code:      r.Code(),
		Token:     r.Message.Token(),
		Options:   r.Message.Options(),
		MessageID: r.MessageID(),
		Type:      r.typ,
	}
	payload, err := r.ReadBody()
	if err != nil {
		return nil, err
	}
	m.Payload = payload
	size, err := m.Size()
	if err != nil {
		return nil, err
	}
	if len(r.rawMarshalData) < size {
		r.rawMarshalData = append(r.rawMarshalData, make([]byte, size-len(r.rawMarshalData))...)
	}
	n, err := m.MarshalTo(r.rawMarshalData)
	if err != nil {
		return nil, err
	}
	r.rawMarshalData = r.rawMarshalData[:n]
	return r.rawMarshalData, nil
}

func (r *Message) IsSeparate() bool {
	return r.Code() == codes.Empty && r.Token() == nil && r.Type() == udp.Acknowledgement && len(r.Options()) == 0 && r.Body() == nil
}

func (r *Message) String() string {
	if r == nil {
		return "nil"
	}
	mid := "nil"
	if r.messageID != nil {
		mid = fmt.Sprint(*r.messageID)
	}
	return fmt.Sprintf("Type: %v, MID: %v, %s", r.Type(), mid, r.Message.String())
}

type Pool struct {
	// This field needs to be the first in the struct to ensure proper word alignment on 32-bit platforms.
	// See: https://golang.org/pkg/sync/atomic/#pkg-note-BUG
	currentMessagesInPool int64
	messagePool           sync.Pool
	maxNumMessages        uint32
	maxMessageBufferSize  uint16
}

func New(maxNumMessages uint32, maxMessageBufferSize uint16) *Pool {
	return &Pool{
		maxNumMessages:       maxNumMessages,
		maxMessageBufferSize: maxMessageBufferSize,
	}
}

// AcquireMessage returns an empty Message instance from Message pool.
//
// The returned Message instance may be passed to ReleaseMessage when it is
// no longer needed. This allows Message recycling, reduces GC pressure
// and usually improves performance.
func (p *Pool) AcquireMessage(ctx context.Context) *Message {
	v := p.messagePool.Get()
	if v == nil {
		return &Message{
			Message:        pool.NewMessage(),
			rawData:        make([]byte, 256),
			rawMarshalData: make([]byte, 256),
			ctx:            ctx,
		}
	}
	r := v.(*Message)
	atomic.AddInt64(&p.currentMessagesInPool, -1)
	r.ctx = ctx
	return r
}

// ReleaseMessage returns req acquired via AcquireMessage to Message pool.
//
// It is forbidden accessing req and/or its' members after returning
// it to Message pool.
func (p *Pool) ReleaseMessage(req *Message) {
	v := atomic.LoadInt64(&p.currentMessagesInPool)
	if v >= int64(p.maxNumMessages) {
		return
	}
	atomic.AddInt64(&p.currentMessagesInPool, 1)
	req.Reset()
	req.ctx = nil
	p.messagePool.Put(req)
}

// ConvertFrom converts common message to pool message.
func (p *Pool) ConvertFrom(m *message.Message) (*Message, error) {
	if m.Context == nil {
		return nil, fmt.Errorf("invalid context")
	}
	r := p.AcquireMessage(m.Context)
	r.SetCode(m.Code)
	r.ResetOptionsTo(m.Options)
	r.SetBody(m.Body)
	r.SetToken(m.Token)
	return r, nil
}

// ConvertTo converts pool message to common message.
func ConvertTo(m *Message) (*message.Message, error) {
	opts, err := m.Options().Clone()
	if err != nil {
		return nil, err
	}
	var body io.ReadSeeker
	if m.Body() != nil {
		payload, err := m.ReadBody()
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(payload)
	}
	return &message.Message{
		Context: m.Context(),
		Code:    m.Code(),
		Token:   m.Token(),
		Body:    body,
		Options: opts,
	}, nil
}
