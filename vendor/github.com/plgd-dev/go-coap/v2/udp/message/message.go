package message

import (
	"encoding/binary"
	"errors"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

// TcpMessage is a CoAP MessageBase that can encode itself for Message
// transport.
type Message struct {
	Token   message.Token
	Payload []byte

	Options message.Options //Options must be sorted by ID
	Code    codes.Code

	MessageID uint16
	Type      Type
}

func (m Message) Size() (int, error) {
	if len(m.Token) > message.MaxTokenSize {
		return -1, message.ErrInvalidTokenLen
	}
	size := 4 + len(m.Token)
	payloadLen := len(m.Payload)
	optionsLen, err := m.Options.Marshal(nil)
	if !errors.Is(err, message.ErrTooSmall) {
		return -1, err
	}
	if payloadLen > 0 {
		//for separator 0xff
		payloadLen++
	}
	size += payloadLen + optionsLen
	return size, nil
}

func (m Message) Marshal() ([]byte, error) {
	b := make([]byte, 1024)
	l, err := m.MarshalTo(b)
	if errors.Is(err, message.ErrTooSmall) {
		b = append(b[:0], make([]byte, l)...)
		l, err = m.MarshalTo(b)
	}
	return b[:l], err
}

func (m Message) MarshalTo(buf []byte) (int, error) {
	/*
	     0                   1                   2                   3
	    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |Ver| T |  TKL  |      Code     |          Message ID           |
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |   Token (if any, TKL bytes) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |   Options (if any) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	   |1 1 1 1 1 1 1 1|    Payload (if any) ...
	   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/
	size, err := m.Size()
	if err != nil {
		return -1, err
	}
	if len(buf) < size {
		return size, message.ErrTooSmall
	}

	tmpbuf := []byte{0, 0}
	binary.BigEndian.PutUint16(tmpbuf, m.MessageID)

	buf[0] = (1 << 6) | byte(m.Type)<<4 | byte(0xf&len(m.Token))
	buf[1] = byte(m.Code)
	buf[2] = tmpbuf[0]
	buf[3] = tmpbuf[1]
	buf = buf[4:]

	if len(m.Token) > message.MaxTokenSize {
		return -1, message.ErrInvalidTokenLen
	}
	copy(buf, m.Token)
	buf = buf[len(m.Token):]

	optionsLen, err := m.Options.Marshal(buf)
	switch {
	case err == nil:
	case errors.Is(err, message.ErrTooSmall):
		return size, err
	default:
		return -1, err
	}
	buf = buf[optionsLen:]

	if len(m.Payload) > 0 {
		buf[0] = 0xff
		buf = buf[1:]
	}
	copy(buf, m.Payload)
	return size, nil
}

func (m *Message) Unmarshal(data []byte) (int, error) {
	size := len(data)
	if size < 4 {
		return -1, ErrMessageTruncated
	}

	if data[0]>>6 != 1 {
		return -1, ErrMessageInvalidVersion
	}

	typ := Type((data[0] >> 4) & 0x3)
	tokenLen := int(data[0] & 0xf)
	if tokenLen > 8 {
		return -1, message.ErrInvalidTokenLen
	}

	code := codes.Code(data[1])
	messageID := binary.BigEndian.Uint16(data[2:4])
	data = data[4:]
	if len(data) < tokenLen {
		return -1, ErrMessageTruncated
	}
	token := data[:tokenLen]
	if len(token) == 0 {
		token = nil
	}
	data = data[tokenLen:]

	optionDefs := message.CoapOptionDefs
	proc, err := m.Options.Unmarshal(data, optionDefs)
	if err != nil {
		return -1, err
	}
	data = data[proc:]
	if len(data) == 0 {
		data = nil
	}

	m.Payload = data
	m.Code = code
	m.Token = token
	m.Type = typ
	m.MessageID = messageID

	return size, nil
}
