// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdufield

import (
	"bytes"
	"fmt"
	"io"
)

// List is a list of PDU fields.
type List []Name

// Decode decodes binary data in the given buffer to build a Map.
//
// If the ShortMessage field is present, and DataCoding as well,
// we attempt to decode text automatically. See pdutext package
// for more information.
func (l List) Decode(r *bytes.Buffer) (Map, error) {
	var (
		unsuccessCount, numDest, udhLength, smLength int

		udhiFlag bool
	)
	f := make(Map)
loop:
	for _, k := range l {
		switch k {
		case
			AddressRange,
			DestinationAddr,
			ErrorCode,
			FinalDate,
			MessageID,
			MessageState,
			Password,
			ScheduleDeliveryTime,
			ServiceType,
			SourceAddr,
			SystemID,
			SystemType,
			ValidityPeriod:
			b, err := r.ReadBytes(0x00)
			if err == io.EOF {
				break loop
			}
			if err != nil {
				return nil, err
			}
			f[k] = &Variable{Data: b}
		case
			AddrNPI,
			AddrTON,
			DataCoding,
			DestAddrNPI,
			DestAddrTON,
			ESMClass,
			InterfaceVersion,
			NumberDests,
			NoUnsuccess,
			PriorityFlag,
			ProtocolID,
			RegisteredDelivery,
			ReplaceIfPresentFlag,
			SMDefaultMsgID,
			SourceAddrNPI,
			SourceAddrTON,
			SMLength:
			b, err := r.ReadByte()
			if err == io.EOF {
				break loop
			}
			if err != nil {
				return nil, err
			}
			f[k] = &Fixed{Data: b}
			switch k {
			case NoUnsuccess:
				unsuccessCount = int(b)
			case NumberDests:
				numDest = int(b)
			case SMLength:
				smLength = int(b)
			case ESMClass:
				mask := byte(1 << 6)
				udhiFlag = mask == b&mask
			}
		case UDHLength:
			if !udhiFlag {
				continue
			}
			b, err := r.ReadByte()
			if err == io.EOF {
				break loop
			}
			if err != nil {
				return nil, err
			}
			udhLength = int(b)
			f[k] = &Fixed{Data: b}
		case GSMUserData:
			if !udhiFlag {
				continue
			}
			var udhList []UDH
			var l int
			for i := udhLength; i > 0; i -= l + 2 {
				var udh UDH
				// Read IEI
				b, err := r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				udh.IEI = Fixed{Data: b}
				// Read IELength
				b, err = r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				l = int(b)
				udh.IELength = Fixed{Data: b}
				// Read IEData
				bt := r.Next(l)
				udh.IEData = Variable{Data: bt}
				udhList = append(udhList, udh)
				if len(bt) != l {
					break loop
				}
			}
			f[k] = &UDHList{Data: udhList}
		case DestinationList:
			var destList []DestSme
			for i := 0; i < numDest; i++ {
				var dest DestSme
				// Read DestFlag
				b, err := r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				dest.Flag = Fixed{Data: b}
				// Read Ton
				b, err = r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				dest.Ton = Fixed{Data: b}
				// Read npi
				b, err = r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				dest.Npi = Fixed{Data: b}
				// Read address
				bt, err := r.ReadBytes(0x00)
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				dest.DestAddr = Variable{Data: bt}
				destList = append(destList, dest)
			}
			f[k] = &DestSmeList{Data: destList}
		case UnsuccessSme:
			var unsList []UnSme
			for i := 0; i < unsuccessCount; i++ {
				var uns UnSme
				// Read Ton
				b, err := r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				uns.Ton = Fixed{Data: b}
				// Read npi
				b, err = r.ReadByte()
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				uns.Npi = Fixed{Data: b}
				// Read address
				bt, err := r.ReadBytes(0x00)
				if err == io.EOF {
					break loop
				}
				if err != nil {
					return nil, err
				}
				uns.DestAddr = Variable{Data: bt}
				// Read error code
				uns.ErrCode = Variable{Data: r.Next(4)}
				// Add unSme to the list
				unsList = append(unsList, uns)
			}
			f[k] = &UnSmeList{Data: unsList}
		case ShortMessage:
			// Check UDHLength
			if udhLength > 0 {
				if smLength-udhLength-1 < 0 {
					return nil, fmt.Errorf("smLength is lesser than udhLength+1: have %d and %d",
						smLength, udhLength)
				}
				smLength -= udhLength + 1
				f[SMLength] = &Fixed{Data: byte(smLength)}
			}
			// Check SMLength
			if r.Len() < smLength {
				return nil, fmt.Errorf("short read for smlength: want %d, have %d",
					smLength, r.Len())
			}
			f[ShortMessage] = &SM{Data: r.Next(smLength)}
		}
	}
	return f, nil
}
