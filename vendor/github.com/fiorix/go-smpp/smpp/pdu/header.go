// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import (
	"encoding/binary"
	"fmt"
	"io"
)

type (
	// ID of the PDU header.
	ID uint32

	// Status is a property of the PDU header.
	Status uint32
)

var idString = map[ID]string{
	GenericNACKID:         "GenericNACK",
	BindReceiverID:        "BindReceiver",
	BindReceiverRespID:    "BindReceiverResp",
	BindTransmitterID:     "BindTransmitter",
	BindTransmitterRespID: "BindTransmitterResp",
	QuerySMID:             "QuerySM",
	QuerySMRespID:         "QuerySMResp",
	SubmitSMID:            "SubmitSM",
	SubmitSMRespID:        "SubmitSMResp",
	DeliverSMID:           "DeliverSM",
	DeliverSMRespID:       "DeliverSMResp",
	UnbindID:              "Unbind",
	UnbindRespID:          "UnbindResp",
	ReplaceSMID:           "ReplaceSM",
	ReplaceSMRespID:       "ReplaceSMResp",
	CancelSMID:            "CancelSM",
	CancelSMRespID:        "CancelSMResp",
	BindTransceiverID:     "BindTransceiver",
	BindTransceiverRespID: "BindTransceiverResp",
	OutbindID:             "Outbind",
	EnquireLinkID:         "EnquireLink",
	EnquireLinkRespID:     "EnquireLinkResp",
	SubmitMultiID:         "SubmitMulti",
	SubmitMultiRespID:     "SubmitMultiResp",
	AlertNotificationID:   "AlertNotification",
	DataSMID:              "DataSM",
	DataSMRespID:          "DataSMResp",
}

// String returns the PDU type as a string.
func (id ID) String() string {
	return idString[id]
}

// HeaderLen is the PDU header length.
const HeaderLen = 16

// Header is a PDU header.
type Header struct {
	Len    uint32
	ID     ID
	Status Status
	Seq    uint32 // Sequence number.
}

// DecodeHeader decodes binary PDU header data.
func DecodeHeader(r io.Reader) (*Header, error) {
	b := make([]byte, HeaderLen)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	l := binary.BigEndian.Uint32(b[0:4])
	if l < HeaderLen {
		return nil, fmt.Errorf("PDU too small: %d < %d", l, HeaderLen)
	}
	if l > MaxSize {
		return nil, fmt.Errorf("PDU too large: %d > %d", l, MaxSize)
	}
	hdr := &Header{
		Len:    l,
		ID:     ID(binary.BigEndian.Uint32(b[4:8])),
		Status: Status(binary.BigEndian.Uint32(b[8:12])),
		Seq:    binary.BigEndian.Uint32(b[12:16]),
	}
	return hdr, nil
}

// SerializeTo serializes the Header to its binary form to the given writer.
func (h *Header) SerializeTo(w io.Writer) error {
	b := make([]byte, HeaderLen)
	binary.BigEndian.PutUint32(b[0:4], h.Len)
	binary.BigEndian.PutUint32(b[4:8], uint32(h.ID))
	binary.BigEndian.PutUint32(b[8:12], uint32(h.Status))
	binary.BigEndian.PutUint32(b[12:16], h.Seq)
	_, err := w.Write(b)
	return err
}

// Error implements the Error interface.
func (s Status) Error() string {
	m, ok := esmeStatus[s]
	if !ok {
		return fmt.Sprintf("unknown status: %d", s)
	}
	return m
}

var esmeStatus = map[Status]string{
	0x00000000: "OK",
	0x00000001: "invalid message length",
	0x00000002: "invalid command length",
	0x00000003: "invalid command id",
	0x00000004: "incorrect bind status for given command",
	0x00000005: "already in bound state",
	0x00000006: "invalid priority flag",
	0x00000007: "invalid registered delivery flag",
	0x00000008: "system error",
	0x0000000a: "invalid source address",
	0x0000000b: "invalid destination address",
	0x0000000c: "invalid message id",
	0x0000000d: "bind failed",
	0x0000000e: "invalid password",
	0x0000000f: "invalid system id",
	0x00000011: "cancelsm failed",
	0x00000013: "replacesm failed",
	0x00000014: "message queue full",
	0x00000015: "invalid service type",
	0x00000033: "invalid number of destinations",
	0x00000034: "invalid distribution list name",
	0x00000040: "invalid destination flag",
	0x00000042: "invalid 'submit with replace' request",
	0x00000043: "invalid esm class field data",
	0x00000044: "cannot submit to distribution list",
	0x00000045: "submitsm or submitmulti failed",
	0x00000048: "invalid source address ton",
	0x00000049: "invalid source address npi",
	0x00000050: "invalid destination address ton",
	0x00000051: "invalid destination address npi",
	0x00000053: "invalid system type field",
	0x00000054: "invalid replace_if_present flag",
	0x00000055: "invalid number of messages",
	0x00000058: "throttling error",
	0x00000061: "invalid scheduled delivery time",
	0x00000062: "invalid message validity period (expiry time)",
	0x00000063: "predefined message invalid or not found",
	0x00000064: "esme receiver temporary app error code",
	0x00000065: "esme receiver permanent app error code",
	0x00000066: "esme receiver reject message error code",
	0x00000067: "querysm request failed",
	0x000000c0: "error in the optional part of the pdu body",
	0x000000c1: "optional parameter not allowed",
	0x000000c2: "invalid parameter length",
	0x000000c3: "expected optional parameter missing",
	0x000000c4: "invalid optional parameter value",
	0x000000fe: "delivery failure (used for datasmresp)",
	0x000000ff: "unknown error",
}
