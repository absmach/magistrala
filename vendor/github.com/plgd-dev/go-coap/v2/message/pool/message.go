package pool

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

var (
	messagePool sync.Pool
)

type Message struct {
	msg             message.Message
	valueBuffer     []byte
	origValueBuffer []byte

	payload    io.ReadSeeker
	sequence   uint64
	hijacked   uint32
	isModified bool
}

func NewMessage() *Message {
	valueBuffer := make([]byte, 256)
	return &Message{
		msg: message.Message{
			Options: make(message.Options, 0, 16),
		},
		valueBuffer:     valueBuffer,
		origValueBuffer: valueBuffer,
	}
}

// Reset clear message for next reuse
func (r *Message) Reset() {
	r.msg.Token = nil
	r.msg.Code = codes.Empty
	r.msg.Options = r.msg.Options[:0]
	r.valueBuffer = r.origValueBuffer
	r.payload = nil
	r.isModified = false
}

func (r *Message) Remove(opt message.OptionID) {
	r.msg.Options = r.msg.Options.Remove(opt)
	r.isModified = true
}

func (r *Message) Token() message.Token {
	if r.msg.Token == nil {
		return nil
	}
	token := make(message.Token, 0, 8)
	token = append(token, r.msg.Token...)
	return token
}

func (r *Message) SetToken(token message.Token) {
	if token == nil {
		r.msg.Token = nil
		return
	}
	r.msg.Token = append(r.msg.Token[:0], token...)
}

func (r *Message) ResetOptionsTo(in message.Options) {
	opts, used, err := r.msg.Options.ResetOptionsTo(r.valueBuffer, in)
	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.ResetOptionsTo(r.valueBuffer, in)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	if len(in) > 0 {
		r.isModified = true
	}
}

func (r *Message) Options() message.Options {
	return r.msg.Options
}

func (r *Message) SetPath(p string) {
	opts, used, err := r.msg.Options.SetPath(r.valueBuffer, p)

	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.SetPath(r.valueBuffer, p)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	r.isModified = true
}

func (r *Message) Code() codes.Code {
	return r.msg.Code
}

func (r *Message) SetCode(code codes.Code) {
	r.msg.Code = code
	r.isModified = true
}

func (r *Message) SetETag(value []byte) {
	r.SetOptionBytes(message.ETag, value)
}

func (r *Message) GetETag() ([]byte, error) {
	return r.GetOptionBytes(message.ETag)
}

func (r *Message) AddQuery(query string) {
	r.AddOptionString(message.URIQuery, query)
}

func (r *Message) GetOptionUint32(id message.OptionID) (uint32, error) {
	return r.msg.Options.GetUint32(id)
}

func (r *Message) SetOptionString(opt message.OptionID, value string) {
	opts, used, err := r.msg.Options.SetString(r.valueBuffer, opt, value)
	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.SetString(r.valueBuffer, opt, value)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	r.isModified = true
}

func (r *Message) AddOptionString(opt message.OptionID, value string) {
	opts, used, err := r.msg.Options.AddString(r.valueBuffer, opt, value)
	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.AddString(r.valueBuffer, opt, value)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	r.isModified = true
}

func (r *Message) AddOptionBytes(opt message.OptionID, value []byte) {
	if len(r.valueBuffer) < len(value) {
		r.valueBuffer = append(r.valueBuffer, make([]byte, len(value)-len(r.valueBuffer))...)
	}
	n := copy(r.valueBuffer, value)
	v := r.valueBuffer[:n]
	r.msg.Options = r.msg.Options.Add(message.Option{opt, v})
	r.valueBuffer = r.valueBuffer[n:]
	r.isModified = true
}

func (r *Message) SetOptionBytes(opt message.OptionID, value []byte) {
	if len(r.valueBuffer) < len(value) {
		r.valueBuffer = append(r.valueBuffer, make([]byte, len(value)-len(r.valueBuffer))...)
	}
	n := copy(r.valueBuffer, value)
	v := r.valueBuffer[:n]
	r.msg.Options = r.msg.Options.Set(message.Option{opt, v})
	r.valueBuffer = r.valueBuffer[n:]
	r.isModified = true
}

func (r *Message) GetOptionBytes(id message.OptionID) ([]byte, error) {
	return r.msg.Options.GetBytes(id)
}

func (r *Message) SetOptionUint32(opt message.OptionID, value uint32) {
	opts, used, err := r.msg.Options.SetUint32(r.valueBuffer, opt, value)
	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.SetUint32(r.valueBuffer, opt, value)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	r.isModified = true
}

func (r *Message) AddOptionUint32(opt message.OptionID, value uint32) {
	opts, used, err := r.msg.Options.AddUint32(r.valueBuffer, opt, value)
	if err == message.ErrTooSmall {
		r.valueBuffer = append(r.valueBuffer, make([]byte, used)...)
		opts, used, err = r.msg.Options.AddUint32(r.valueBuffer, opt, value)
	}
	r.msg.Options = opts
	r.valueBuffer = r.valueBuffer[used:]
	r.isModified = true
}

func (r *Message) ContentFormat() (message.MediaType, error) {
	v, err := r.GetOptionUint32(message.ContentFormat)
	return message.MediaType(v), err
}

func (r *Message) HasOption(id message.OptionID) bool {
	return r.msg.Options.HasOption(id)
}

func (r *Message) SetContentFormat(contentFormat message.MediaType) {
	r.SetOptionUint32(message.ContentFormat, uint32(contentFormat))
}

func (r *Message) SetObserve(observe uint32) {
	r.SetOptionUint32(message.Observe, observe)
}

func (r *Message) Observe() (uint32, error) {
	return r.GetOptionUint32(message.Observe)
}

// SetAccept set's accept option.
func (r *Message) SetAccept(contentFormat message.MediaType) {
	r.SetOptionUint32(message.Accept, uint32(contentFormat))
}

// Accept get's accept option.
func (r *Message) Accept() (message.MediaType, error) {
	v, err := r.GetOptionUint32(message.Accept)
	return message.MediaType(v), err
}

func (r *Message) ETag() ([]byte, error) {
	return r.GetOptionBytes(message.ETag)
}

func (r *Message) BodySize() (int64, error) {
	if r.payload == nil {
		return 0, nil
	}
	orig, err := r.payload.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	_, err = r.payload.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	size, err := r.payload.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = r.payload.Seek(orig, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (r *Message) SetBody(s io.ReadSeeker) {
	r.payload = s
	r.isModified = true
}

func (r *Message) Body() io.ReadSeeker {
	return r.payload
}

func (r *Message) SetSequence(seq uint64) {
	r.sequence = seq
}

func (r *Message) Sequence() uint64 {
	return r.sequence
}

func (r *Message) Hijack() {
	atomic.StoreUint32(&r.hijacked, 1)
}

func (r *Message) IsHijacked() bool {
	return atomic.LoadUint32(&r.hijacked) == 1
}

func (r *Message) IsModified() bool {
	return r.isModified
}

func (r *Message) String() string {
	return r.msg.String()
}

func (r *Message) ReadBody() ([]byte, error) {
	if r.Body() == nil {
		return nil, nil
	}
	payload := make([]byte, 1024)
	size, err := r.BodySize()
	if err != nil {
		return nil, err
	}
	_, err = r.Body().Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	if int64(len(payload)) < size {
		payload = make([]byte, size)
	}
	n, err := io.ReadFull(r.Body(), payload)
	if err == io.ErrUnexpectedEOF && int64(n) == size {
		err = nil
	} else if err != nil {
		return nil, err
	}
	return payload[:n], nil
}
