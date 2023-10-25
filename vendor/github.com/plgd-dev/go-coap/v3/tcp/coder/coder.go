package coder

import (
	"encoding/binary"
	"errors"

	"github.com/plgd-dev/go-coap/v3/message"
	"github.com/plgd-dev/go-coap/v3/message/codes"
)

var DefaultCoder = new(Coder)

const (
	MessageLength13Base = 13
	MessageLength14Base = 269
	MessageLength15Base = 65805
	messageMaxLen       = 0x7fff0000 // Large number that works in 32-bit builds
)

type Coder struct{}

type MessageHeader struct {
	Token         []byte
	Length        uint32
	MessageLength uint32
	Code          codes.Code
}

func (c *Coder) Size(m message.Message) (int, error) {
	size, err := c.Encode(m, nil)
	if errors.Is(err, message.ErrTooSmall) {
		err = nil
	}
	return size, err
}

func getHeader(messageLength int) (uint8, []byte) {
	if messageLength < MessageLength13Base {
		return uint8(messageLength), nil
	}
	if messageLength < MessageLength14Base {
		extLen := messageLength - MessageLength13Base
		extLenBytes := []byte{uint8(extLen)}
		return 13, extLenBytes
	}
	if messageLength < MessageLength15Base {
		extLen := messageLength - MessageLength14Base
		extLenBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(extLenBytes, uint16(extLen))
		return 14, extLenBytes
	}
	if messageLength < messageMaxLen {
		extLen := messageLength - MessageLength15Base
		extLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(extLenBytes, uint32(extLen))
		return 15, extLenBytes
	}
	return 0, nil
}

func (c *Coder) Encode(m message.Message, buf []byte) (int, error) {
	/*
	   A CoAP Message message lomessage.OKs like:

	        0                   1                   2                   3
	       0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
	      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	      |  Len  |  TKL  | Extended Length ...
	      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	      |      Code     | TKL bytes ...
	      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	      |   Options (if any) ...
	      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	      |1 1 1 1 1 1 1 1|    Payload (if any) ...
	      +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

	   The size of the Extended Length field is inferred from the value of the
	   Len field as follows:

	   | Len value  | Extended Length size  | Total length              |
	   +------------+-----------------------+---------------------------+
	   | 0-12       | 0                     | Len                       |
	   | 13         | 1                     | Extended Length + 13      |
	   | 14         | 2                     | Extended Length + 269     |
	   | 15         | 4                     | Extended Length + 65805   |
	*/

	if len(m.Token) > message.MaxTokenSize {
		return -1, message.ErrInvalidTokenLen
	}

	payloadLen := len(m.Payload)
	if payloadLen > 0 {
		// for separator 0xff
		payloadLen++
	}
	optionsLen, err := m.Options.Marshal(nil)
	if !errors.Is(err, message.ErrTooSmall) {
		return -1, err
	}
	bufLen := payloadLen + optionsLen
	lenNib, extLenBytes := getHeader(bufLen)

	var hdr [1 + 4 + message.MaxTokenSize + 1]byte
	hdrLen := 1 + len(extLenBytes) + len(m.Token) + 1
	hdrOff := 0

	copyToHdr := func(offset int, data []byte) int {
		if len(data) > 0 {
			copy(hdr[hdrOff:hdrOff+len(data)], data)
			offset += len(data)
		}
		return offset
	}

	// Length and TKL nibbles.
	hdr[hdrOff] = uint8(0xf&len(m.Token)) | (lenNib << 4)
	hdrOff++

	// Extended length, if present.
	hdrOff = copyToHdr(hdrOff, extLenBytes)

	// Code.
	hdr[hdrOff] = byte(m.Code)
	hdrOff++

	// Token.
	copyToHdr(hdrOff, m.Token)

	bufLen += hdrLen
	if len(buf) < bufLen {
		return bufLen, message.ErrTooSmall
	}

	copy(buf, hdr[:hdrLen])
	optionsLen, err = m.Options.Marshal(buf[hdrLen:])
	switch {
	case err == nil:
	case errors.Is(err, message.ErrTooSmall):
		return bufLen, err
	default:
		return -1, err
	}
	if len(m.Payload) > 0 {
		copy(buf[hdrLen+optionsLen:], []byte{0xff})
		copy(buf[hdrLen+optionsLen+1:], m.Payload)
	}

	return bufLen, nil
}

func (c *Coder) DecodeHeader(data []byte, h *MessageHeader) (int, error) {
	hdrOff := uint32(0)
	if len(data) == 0 {
		return -1, message.ErrShortRead
	}

	firstByte := data[0]
	data = data[1:]
	hdrOff++

	lenNib := (firstByte & 0xf0) >> 4
	tkl := firstByte & 0x0f

	var opLen int
	switch {
	case lenNib < MessageLength13Base:
		opLen = int(lenNib)
	case lenNib == 13:
		if len(data) < 1 {
			return -1, message.ErrShortRead
		}
		extLen := data[0]
		data = data[1:]
		hdrOff++
		opLen = MessageLength13Base + int(extLen)
	case lenNib == 14:
		if len(data) < 2 {
			return -1, message.ErrShortRead
		}
		extLen := binary.BigEndian.Uint16(data)
		data = data[2:]
		hdrOff += 2
		opLen = MessageLength14Base + int(extLen)
	case lenNib == 15:
		if len(data) < 4 {
			return -1, message.ErrShortRead
		}
		extLen := binary.BigEndian.Uint32(data)
		data = data[4:]
		hdrOff += 4
		opLen = MessageLength15Base + int(extLen)
	}

	h.MessageLength = hdrOff + 1 + uint32(tkl) + uint32(opLen)
	if len(data) < 1 {
		return -1, message.ErrShortRead
	}
	h.Code = codes.Code(data[0])
	data = data[1:]
	hdrOff++
	if len(data) < int(tkl) {
		return -1, message.ErrShortRead
	}
	if tkl > 0 {
		h.Token = data[:tkl]
	}
	hdrOff += uint32(tkl)
	h.Length = hdrOff
	return int(h.Length), nil
}

func (c *Coder) DecodeWithHeader(data []byte, header MessageHeader, m *message.Message) (int, error) {
	optionDefs := message.CoapOptionDefs
	processed := header.Length
	switch header.Code {
	case codes.CSM:
		optionDefs = message.TCPSignalCSMOptionDefs
	case codes.Ping, codes.Pong:
		optionDefs = message.TCPSignalPingPongOptionDefs
	case codes.Release:
		optionDefs = message.TCPSignalReleaseOptionDefs
	case codes.Abort:
		optionDefs = message.TCPSignalAbortOptionDefs
	}

	proc, err := m.Options.Unmarshal(data, optionDefs)
	if err != nil {
		return -1, err
	}
	data = data[proc:]
	processed += uint32(proc)

	if len(data) > 0 {
		m.Payload = data
	}
	processed += uint32(len(data))
	m.Code = header.Code
	m.Token = header.Token

	return int(processed), nil
}

func (c *Coder) Decode(data []byte, m *message.Message) (int, error) {
	var header MessageHeader
	_, err := c.DecodeHeader(data, &header)
	if err != nil {
		return -1, err
	}
	if uint32(len(data)) < header.MessageLength {
		return -1, message.ErrShortRead
	}
	return c.DecodeWithHeader(data[header.Length:], header, m)
}
