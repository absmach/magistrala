package coder

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

var DefaultCoder = new(Coder)

type Coder struct{}

func (c *Coder) Size(m message.Message) (int, error) {
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
		// for separator 0xff
		payloadLen++
	}
	size += payloadLen + optionsLen
	return size, nil
}

func (c *Coder) Encode(m message.Message, buf []byte) (int, error) {
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
	if !message.ValidateMID(m.MessageID) {
		return -1, fmt.Errorf("invalid MessageID(%v)", m.MessageID)
	}
	if !message.ValidateType(m.Type) {
		return -1, fmt.Errorf("invalid Type(%v)", m.Type)
	}
	size, err := c.Size(m)
	if err != nil {
		return -1, err
	}
	if len(buf) < size {
		return size, message.ErrTooSmall
	}

	tmpbuf := []byte{0, 0}
	binary.BigEndian.PutUint16(tmpbuf, uint16(m.MessageID))

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

func (c *Coder) Decode(data []byte, m *message.Message) (int, error) {
	size := len(data)
	if size < 4 {
		return -1, ErrMessageTruncated
	}

	if data[0]>>6 != 1 {
		return -1, ErrMessageInvalidVersion
	}

	typ := message.Type((data[0] >> 4) & 0x3)
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
	m.MessageID = int32(messageID)

	return size, nil
}
