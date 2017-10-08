package coap

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestTCPDecodeMessageSmallWithPayload(t *testing.T) {
	input := []byte{0, 0,
		0x40, 0x1, 0x30, 0x39, 0x21, 0x3,
		0x26, 0x77, 0x65, 0x65, 0x74, 0x61, 0x67,
		0xff, 'h', 'i',
	}

	binary.BigEndian.PutUint16(input, uint16(len(input)-2))

	msg, err := Decode(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Error parsing message: %v", err)
	}

	if msg.Type != Confirmable {
		t.Errorf("Expected message type confirmable, got %v", msg.Type)
	}
	if msg.Code != GET {
		t.Errorf("Expected message code GET, got %v", msg.Code)
	}
	if msg.MessageID != 12345 {
		t.Errorf("Expected message ID 12345, got %v", msg.MessageID)
	}

	if !bytes.Equal(msg.Payload, []byte("hi")) {
		t.Errorf("Incorrect payload: %q", msg.Payload)
	}
}
