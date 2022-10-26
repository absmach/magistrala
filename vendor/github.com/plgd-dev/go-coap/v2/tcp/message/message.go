package message

import (
	"encoding/binary"
	"errors"

	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

const (
	messageLen13Base = 13
	messageLen14Base = 269
	messageLen15Base = 65805
	messageMaxLen    = 0x7fff0000 // Large number that works in 32-bit builds
)

// Signal CSM Option IDs
/*
   +-----+---+---+-------------------+--------+--------+---------+
   | No. | C | R | Name              | Format | Length | Default |
   +-----+---+---+-------------------+--------+--------+---------+
   |   2 |   |   | MaxMessageSize    | uint   | 0-4    | 1152    |
   |   4 |   |   | BlockWiseTransfer | empty  | 0      | (none)  |
   +-----+---+---+-------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	MaxMessageSize    message.OptionID = 2
	BlockWiseTransfer message.OptionID = 4
)

// Signal Ping/Pong Option IDs
/*
   +-----+---+---+-------------------+--------+--------+---------+
   | No. | C | R | Name              | Format | Length | Default |
   +-----+---+---+-------------------+--------+--------+---------+
   |   2 |   |   | Custody           | empty  | 0      | (none)  |
   +-----+---+---+-------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	Custody message.OptionID = 2
)

// Signal Release Option IDs
/*
   +-----+---+---+---------------------+--------+--------+---------+
   | No. | C | R | Name                | Format | Length | Default |
   +-----+---+---+---------------------+--------+--------+---------+
   |   2 |   | x | Alternative-Address | string | 1-255  | (none)  |
   |   4 |   |   | Hold-Off            | uint3  | 0-3    | (none)  |
   +-----+---+---+---------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/

const (
	AlternativeAddress message.OptionID = 2
	HoldOff            message.OptionID = 4
)

// Signal Abort Option IDs
/*
   +-----+---+---+---------------------+--------+--------+---------+
   | No. | C | R | Name                | Format | Length | Default |
   +-----+---+---+---------------------+--------+--------+---------+
   |   2 |   |   | Bad-CSM-Option      | uint   | 0-2    | (none)  |
   +-----+---+---+---------------------+--------+--------+---------+
   C=Critical, R=Repeatable
*/
const (
	BadCSMOption message.OptionID = 2
)

var signalCSMOptionDefs = map[message.OptionID]message.OptionDef{
	MaxMessageSize:    {ValueFormat: message.ValueUint, MinLen: 0, MaxLen: 4},
	BlockWiseTransfer: {ValueFormat: message.ValueEmpty, MinLen: 0, MaxLen: 0},
}

var signalPingPongOptionDefs = map[message.OptionID]message.OptionDef{
	Custody: {ValueFormat: message.ValueEmpty, MinLen: 0, MaxLen: 0},
}

var signalReleaseOptionDefs = map[message.OptionID]message.OptionDef{
	AlternativeAddress: {ValueFormat: message.ValueString, MinLen: 1, MaxLen: 255},
	HoldOff:            {ValueFormat: message.ValueUint, MinLen: 0, MaxLen: 3},
}

var signalAbortOptionDefs = map[message.OptionID]message.OptionDef{
	BadCSMOption: {ValueFormat: message.ValueUint, MinLen: 0, MaxLen: 2},
}

// TcpMessage is a CoAP MessageBase that can encode itself for Message
// transport.
type Message struct {
	Token   []byte
	Payload []byte

	Options message.Options //Options must be sorted by ID
	Code    codes.Code
}

func (m Message) Size() (int, error) {
	size, err := m.MarshalTo(nil)
	if errors.Is(err, message.ErrTooSmall) {
		err = nil
	}
	return size, err
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
		//for separator 0xff
		payloadLen++
	}
	optionsLen, err := m.Options.Marshal(nil)
	if !errors.Is(err, message.ErrTooSmall) {
		return -1, err
	}
	bufLen := payloadLen + optionsLen
	var lenNib uint8
	var extLenBytes []byte

	if bufLen < messageLen13Base {
		lenNib = uint8(bufLen)
	} else if bufLen < messageLen14Base {
		lenNib = 13
		extLen := bufLen - messageLen13Base
		extLenBytes = []byte{uint8(extLen)}
	} else if bufLen < messageLen15Base {
		lenNib = 14
		extLen := bufLen - messageLen14Base
		extLenBytes = make([]byte, 2)
		binary.BigEndian.PutUint16(extLenBytes, uint16(extLen))
	} else if bufLen < messageMaxLen {
		lenNib = 15
		extLen := bufLen - messageLen15Base
		extLenBytes = make([]byte, 4)
		binary.BigEndian.PutUint32(extLenBytes, uint32(extLen))
	}

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

	bufLen = bufLen + hdrLen
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

type MessageHeader struct {
	Token     []byte
	HeaderLen uint32
	TotalLen  uint32
	Code      codes.Code
}

// Unmarshal infers information about a Message CoAP message from the first
// fragment.
func (i *MessageHeader) Unmarshal(data []byte) error {
	hdrOff := uint32(0)
	if len(data) == 0 {
		return message.ErrShortRead
	}

	firstByte := data[0]
	data = data[1:]
	hdrOff++

	lenNib := (firstByte & 0xf0) >> 4
	tkl := firstByte & 0x0f

	var opLen int
	switch {
	case lenNib < messageLen13Base:
		opLen = int(lenNib)
	case lenNib == 13:
		if len(data) < 1 {
			return message.ErrShortRead
		}
		extLen := data[0]
		data = data[1:]
		hdrOff++
		opLen = messageLen13Base + int(extLen)
	case lenNib == 14:
		if len(data) < 2 {
			return message.ErrShortRead
		}
		extLen := binary.BigEndian.Uint16(data)
		data = data[2:]
		hdrOff += 2
		opLen = messageLen14Base + int(extLen)
	case lenNib == 15:
		if len(data) < 4 {
			return message.ErrShortRead
		}
		extLen := binary.BigEndian.Uint32(data)
		data = data[4:]
		hdrOff += 4
		opLen = messageLen15Base + int(extLen)
	}

	i.TotalLen = hdrOff + 1 + uint32(tkl) + uint32(opLen)
	if len(data) < 1 {
		return message.ErrShortRead
	}
	i.Code = codes.Code(data[0])
	data = data[1:]
	hdrOff++
	if len(data) < int(tkl) {
		return message.ErrShortRead
	}
	if tkl > 0 {
		i.Token = data[:tkl]
	}
	hdrOff += uint32(tkl)

	i.HeaderLen = hdrOff

	return nil
}

func (m *Message) UnmarshalWithHeader(header MessageHeader, data []byte) (int, error) {
	optionDefs := message.CoapOptionDefs
	processed := header.HeaderLen
	switch header.Code {
	case codes.CSM:
		optionDefs = signalCSMOptionDefs
	case codes.Ping, codes.Pong:
		optionDefs = signalPingPongOptionDefs
	case codes.Release:
		optionDefs = signalReleaseOptionDefs
	case codes.Abort:
		optionDefs = signalAbortOptionDefs
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
	processed = processed + uint32(len(data))
	m.Code = header.Code
	m.Token = header.Token

	return int(processed), nil
}

func (m *Message) Unmarshal(data []byte) (int, error) {
	header := MessageHeader{Token: m.Token}
	err := header.Unmarshal(data)
	if err != nil {
		return -1, err
	}
	if uint32(len(data)) < header.TotalLen {
		return -1, message.ErrShortRead
	}
	return m.UnmarshalWithHeader(header, data[header.HeaderLen:])
}
