package message

import (
	"strconv"
)

// Type represents the message type.
// It's only part of CoAP UDP messages.
// Reliable transports like TCP do not have a type.
type Type uint8

const (
	// Confirmable messages require acknowledgements.
	Confirmable Type = 0
	// NonConfirmable messages do not require acknowledgements.
	NonConfirmable Type = 1
	// Acknowledgement is a message indicating a response to confirmable message.
	Acknowledgement Type = 2
	// Reset indicates a permanent negative acknowledgement.
	Reset Type = 3
)

var typeToString = map[Type]string{
	Confirmable:     "Confirmable",
	NonConfirmable:  "NonConfirmable",
	Acknowledgement: "Acknowledgement",
	Reset:           "reset",
}

func (t Type) String() string {
	val, ok := typeToString[t]
	if ok {
		return val
	}
	return "Type(" + strconv.FormatInt(int64(t), 10) + ")"
}
