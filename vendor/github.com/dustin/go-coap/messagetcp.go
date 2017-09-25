package coap

import (
	"encoding/binary"
	"errors"
	"io"
)

// TcpMessage is a CoAP Message that can encode itself for TCP
// transport.
type TcpMessage struct {
	Message
}

func (m *TcpMessage) MarshalBinary() ([]byte, error) {
	bin, err := m.Message.MarshalBinary()
	if err != nil {
		return nil, err
	}

	/*
		A CoAP TCP message looks like:

		     0                   1                   2                   3
		    0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |        Message Length         |Ver| T |  TKL  |      Code     |
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |   Token (if any, TKL bytes) ...
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |   Options (if any) ...
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		   |1 1 1 1 1 1 1 1|    Payload (if any) ...
		   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	*/

	l := []byte{0, 0}
	binary.BigEndian.PutUint16(l, uint16(len(bin)))

	return append(l, bin...), nil
}

func (m *TcpMessage) UnmarshalBinary(data []byte) error {
	if len(data) < 4 {
		return errors.New("short packet")
	}

	return m.Message.UnmarshalBinary(data)
}

// Decode reads a single message from its input.
func Decode(r io.Reader) (*TcpMessage, error) {
	var ln uint16
	err := binary.Read(r, binary.BigEndian, &ln)
	if err != nil {
		return nil, err
	}

	packet := make([]byte, ln)
	_, err = io.ReadFull(r, packet)
	if err != nil {
		return nil, err
	}

	m := TcpMessage{}

	err = m.UnmarshalBinary(packet)
	return &m, err
}
