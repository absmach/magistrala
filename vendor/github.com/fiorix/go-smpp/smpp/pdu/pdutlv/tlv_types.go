// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdutlv

import (
	"encoding/binary"
	"encoding/hex"
	"io"
)

// Fields is a map of tagged TLV fields
type Fields map[Tag]interface{}

// String is a text string that is not null-terminated.
type String string

// CString is a text string that is automatically null-terminated (e.g., final 00 byte at the end).
type CString string

// Tag is the tag of a Tag-Length-Value (TLV) field.
type Tag uint16

// Hex returns hexadecimal representation of tag
func (t Tag) Hex() string {
	bin := make([]byte, 2, 2)
	binary.BigEndian.PutUint16(bin, uint16(t))
	return hex.EncodeToString(bin)
}

// Common Tag-Length-Value (TLV) tags.
const (
	TagDestAddrSubunit          Tag = 0x0005
	TagDestNetworkType          Tag = 0x0006
	TagDestBearerType           Tag = 0x0007
	TagDestTelematicsID         Tag = 0x0008
	TagSourceAddrSubunit        Tag = 0x000D
	TagSourceNetworkType        Tag = 0x000E
	TagSourceBearerType         Tag = 0x000F
	TagSourceTelematicsID       Tag = 0x0010
	TagQosTimeToLive            Tag = 0x0017
	TagPayloadType              Tag = 0x0019
	TagAdditionalStatusInfoText Tag = 0x001D
	TagReceiptedMessageID       Tag = 0x001E
	TagMsMsgWaitFacilities      Tag = 0x0030
	TagPrivacyIndicator         Tag = 0x0201
	TagSourceSubaddress         Tag = 0x0202
	TagDestSubaddress           Tag = 0x0203
	TagUserMessageReference     Tag = 0x0204
	TagUserResponseCode         Tag = 0x0205
	TagSourcePort               Tag = 0x020A
	TagDestinationPort          Tag = 0x020B
	TagSarMsgRefNum             Tag = 0x020C
	TagLanguageIndicator        Tag = 0x020D
	TagSarTotalSegments         Tag = 0x020E
	TagSarSegmentSeqnum         Tag = 0x020F
	TagCallbackNumPresInd       Tag = 0x0302
	TagCallbackNumAtag          Tag = 0x0303
	TagNumberOfMessages         Tag = 0x0304
	TagCallbackNum              Tag = 0x0381
	TagDpfResult                Tag = 0x0420
	TagSetDpf                   Tag = 0x0421
	TagMsAvailabilityStatus     Tag = 0x0422
	TagNetworkErrorCode         Tag = 0x0423
	TagMessagePayload           Tag = 0x0424
	TagDeliveryFailureReason    Tag = 0x0425
	TagMoreMessagesToSend       Tag = 0x0426
	TagMessageStateOption       Tag = 0x0427
	TagUssdServiceOp            Tag = 0x0501
	TagDisplayTime              Tag = 0x1201
	TagSmsSignal                Tag = 0x1203
	TagMsValidity               Tag = 0x1204
	TagAlertOnMessageDelivery   Tag = 0x130C
	TagItsReplyType             Tag = 0x1380
	TagItsSessionInfo           Tag = 0x1383
)

// Field is a PDU Tag-Length-Value (TLV) field
type Field struct {
	Tag  Tag
	Data []byte
}

// Len implements the Data interface.
func (t *Field) Len() int {
	return len(t.Bytes()) + 4
}

// Raw implements the Data interface.
func (t *Field) Raw() interface{} {
	return t.Bytes()
}

// String implements the Data interface.
func (t *Field) String() string {
	if l := len(t.Data); l > 0 && t.Data[l-1] == 0x00 {
		return string(t.Data[:l-1])
	}
	return string(t.Data)
}

// Bytes implements the Data interface.
func (t *Field) Bytes() []byte {
	return t.Data
}

// SerializeTo implements the Data interface.
func (t *Field) SerializeTo(w io.Writer) error {
	b := make([]byte, len(t.Data)+4)
	binary.BigEndian.PutUint16(b[0:2], uint16(t.Tag))
	binary.BigEndian.PutUint16(b[2:4], uint16(len(t.Data)))
	copy(b[4:], t.Data)

	_, err := w.Write(b)
	return err
}
