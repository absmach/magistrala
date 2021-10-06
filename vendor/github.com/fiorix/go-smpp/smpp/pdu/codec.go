// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdu

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
)

var nextSeq uint32

// codec is the base type of all PDUs.
// It implements the PDU interface and provides a generic encoder.
type codec struct {
	h *Header
	l pdufield.List
	f pdufield.Map
	t pdutlv.Map
}

// init initializes the codec's list and maps and sets the header
// sequence number.
func (pdu *codec) init() {
	if pdu.l == nil {
		pdu.l = pdufield.List{}
	}
	pdu.f = make(pdufield.Map)
	pdu.t = make(pdutlv.Map)
	if pdu.h.Seq == 0 { // If Seq not set
		pdu.h.Seq = atomic.AddUint32(&nextSeq, 1)
	}
}

// setup replaces the codec's current maps with the given ones.
func (pdu *codec) setup(f pdufield.Map, t pdutlv.Map) {
	pdu.f, pdu.t = f, t
}

// Header implements the PDU interface.
func (pdu *codec) Header() *Header {
	return pdu.h
}

// Len implements the PDU interface.
func (pdu *codec) Len() int {
	l := HeaderLen
	for _, f := range pdu.f {
		l += f.Len()
	}
	for _, t := range pdu.t {
		l += t.Len()
	}
	return l
}

// FieldList implements the PDU interface.
func (pdu *codec) FieldList() pdufield.List {
	return pdu.l
}

// Fields implement the PDU interface.
func (pdu *codec) Fields() pdufield.Map {
	return pdu.f
}

// Fields implement the PDU interface.
func (pdu *codec) TLVFields() pdutlv.Map {
	return pdu.t
}

// SerializeTo implements the PDU interface.
func (pdu *codec) SerializeTo(w io.Writer) error {
	var b bytes.Buffer
	for _, k := range pdu.FieldList() {
		f, ok := pdu.f[k]
		if !ok {
			pdu.f.Set(k, nil)
			f = pdu.f[k]
		}
		if err := f.SerializeTo(&b); err != nil {
			return err
		}
	}
	for _, f := range pdu.TLVFields() {
		if err := f.SerializeTo(&b); err != nil {
			return err
		}
	}
	pdu.h.Len = uint32(pdu.Len())
	err := pdu.h.SerializeTo(w)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, &b)
	return err
}

// decoder wraps a PDU (e.g. Bind) and the codec together and is
// used for initializing new PDUs with map data decoded off the wire.
type decoder interface {
	Body
	setup(f pdufield.Map, t pdutlv.Map)
}

func decodeFields(pdu decoder, b []byte) (Body, error) {
	l := pdu.FieldList()
	r := bytes.NewBuffer(b)
	f, err := l.Decode(r)
	if err != nil {
		return nil, err
	}
	t, err := pdutlv.DecodeTLV(r)
	if err != nil {
		return nil, err
	}
	pdu.setup(f, t)
	return pdu, nil
}

// Decode decodes binary PDU data. It returns a new PDU object, e.g. Bind,
// with header and all fields decoded. The returned PDU can be modified
// and re-serialized to its binary form.
func Decode(r io.Reader) (Body, error) {
	hdr, err := DecodeHeader(r)
	if err != nil {
		return nil, err
	}
	b := make([]byte, hdr.Len-HeaderLen)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	switch hdr.ID {
	case AlertNotificationID:
		// TODO(fiorix): Implement AlertNotification.
	case BindReceiverID, BindTransceiverID, BindTransmitterID:
		return decodeFields(newBind(hdr), b)
	case BindReceiverRespID, BindTransceiverRespID, BindTransmitterRespID:
		return decodeFields(newBindResp(hdr), b)
	case CancelSMID:
		// TODO(fiorix): Implement CancelSM.
	case CancelSMRespID:
		// TODO(fiorix): Implement CancelSMResp.
	case DataSMID:
		// TODO(fiorix): Implement DataSM.
	case DataSMRespID:
		// TODO(fiorix): Implement DataSMResp.
	case DeliverSMID:
		return decodeFields(newDeliverSM(hdr), b)
	case DeliverSMRespID:
		return decodeFields(newDeliverSMResp(hdr), b)
	case EnquireLinkID:
		return decodeFields(newEnquireLink(hdr), b)
	case EnquireLinkRespID:
		return decodeFields(newEnquireLinkResp(hdr), b)
	case GenericNACKID:
		return decodeFields(newGenericNACK(hdr), b)
	case OutbindID:
		// TODO(fiorix): Implement Outbind.
	case QuerySMID:
		return decodeFields(newQuerySM(hdr), b)
	case QuerySMRespID:
		return decodeFields(newQuerySMResp(hdr), b)
	case ReplaceSMID:
		// TODO(fiorix): Implement ReplaceSM.
	case ReplaceSMRespID:
		// TODO(fiorix): Implement ReplaceSMResp.
	case SubmitMultiID:
		return decodeFields(newSubmitMulti(hdr), b)
	case SubmitMultiRespID:
		return decodeFields(newSubmitMultiResp(hdr), b)
	case SubmitSMID:
		return decodeFields(newSubmitSM(hdr), b)
	case SubmitSMRespID:
		return decodeFields(newSubmitSMResp(hdr), b)
	case UnbindID:
		return decodeFields(newUnbind(hdr), b)
	case UnbindRespID:
		return decodeFields(newUnbindResp(hdr), b)
	default:
		return nil, fmt.Errorf("unknown PDU type: %#x", hdr.ID)
	}
	return nil, fmt.Errorf("PDU not implemented: %#x", hdr.ID)
}
