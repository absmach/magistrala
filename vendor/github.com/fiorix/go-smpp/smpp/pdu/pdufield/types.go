// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package pdufield

import (
	"encoding/binary"
	"io"
	"strconv"
)

// Name is the name of a PDU field.
type Name string

// Supported PDU field names.
const (
	AddrNPI              Name = "addr_npi"
	AddrTON              Name = "addr_ton"
	AddressRange         Name = "address_range"
	DataCoding           Name = "data_coding"
	DestAddrNPI          Name = "dest_addr_npi"
	DestAddrTON          Name = "dest_addr_ton"
	DestinationAddr      Name = "destination_addr"
	DestinationList      Name = "dest_addresses"
	ESMClass             Name = "esm_class"
	ErrorCode            Name = "error_code"
	FinalDate            Name = "final_date"
	InterfaceVersion     Name = "interface_version"
	MessageID            Name = "message_id"
	MessageState         Name = "message_state"
	NumberDests          Name = "number_of_dests"
	NoUnsuccess          Name = "no_unsuccess"
	Password             Name = "password"
	PriorityFlag         Name = "priority_flag"
	ProtocolID           Name = "protocol_id"
	RegisteredDelivery   Name = "registered_delivery"
	ReplaceIfPresentFlag Name = "replace_if_present_flag"
	SMDefaultMsgID       Name = "sm_default_msg_id"
	SMLength             Name = "sm_length"
	ScheduleDeliveryTime Name = "schedule_delivery_time"
	ServiceType          Name = "service_type"
	ShortMessage         Name = "short_message"
	SourceAddr           Name = "source_addr"
	SourceAddrNPI        Name = "source_addr_npi"
	SourceAddrTON        Name = "source_addr_ton"
	SystemID             Name = "system_id"
	SystemType           Name = "system_type"
	UDHLength            Name = "gsm_sms_ud.udh.len"
	GSMUserData          Name = "gsm_sms_ud.udh"
	UnsuccessSme         Name = "unsuccess_sme"
	ValidityPeriod       Name = "validity_period"
)

// Fixed is a PDU of fixed length.
type Fixed struct {
	Data uint8
}

// Len implements the Data interface.
func (f *Fixed) Len() int {
	return 1
}

// Raw implements the Data interface.
func (f *Fixed) Raw() interface{} {
	return f.Data
}

// String implements the Data interface.
func (f *Fixed) String() string {
	return strconv.Itoa(int(f.Data))
}

// Bytes implements the Data interface.
func (f *Fixed) Bytes() []byte {
	return []byte{f.Data}
}

// SerializeTo implements the Data interface.
func (f *Fixed) SerializeTo(w io.Writer) error {
	_, err := w.Write(f.Bytes())
	return err
}

// Variable is a PDU field of variable length.
type Variable struct {
	Data []byte
}

// Len implements the Data interface.
func (v *Variable) Len() int {
	return len(v.Bytes())
}

// Raw implements the Data interface.
func (v *Variable) Raw() interface{} {
	return v.Data
}

// String implements the Data interface.
func (v *Variable) String() string {
	if l := len(v.Data); l > 0 && v.Data[l-1] == 0x00 {
		return string(v.Data[:l-1])
	}
	return string(v.Data)
}

// Bytes implements the Data interface.
func (v *Variable) Bytes() []byte {
	if len(v.Data) > 0 && v.Data[len(v.Data)-1] == 0x00 {
		return v.Data
	}
	return append(v.Data, 0x00)
}

// SerializeTo implements the Data interface.
func (v *Variable) SerializeTo(w io.Writer) error {
	_, err := w.Write(v.Bytes())
	return err
}

// SM is a PDU field used for Short Messages.
type SM struct {
	Data []byte
}

// Len implements the Data interface.
func (sm *SM) Len() int {
	return len(sm.Data)
}

// Raw implements the Data interface.
func (sm *SM) Raw() interface{} {
	return sm.Data
}

// String implements the Data interface.
func (sm *SM) String() string {
	return string(sm.Data)
}

// Bytes implements the Data interface.
func (sm *SM) Bytes() []byte {
	return sm.Data
}

// SerializeTo implements the Data interface.
func (sm *SM) SerializeTo(w io.Writer) error {
	_, err := w.Write(sm.Bytes())
	return err
}

// DeliverySetting is used to configure registered delivery
// for short messages.
type DeliverySetting uint8

// Supported delivery settings.
const (
	NoDeliveryReceipt      DeliverySetting = 0x00
	FinalDeliveryReceipt   DeliverySetting = 0x01
	FailureDeliveryReceipt DeliverySetting = 0x02
)

// DestSme is a PDU field used for an sme addreses.
type DestSme struct {
	Flag     Fixed
	Ton      Fixed
	Npi      Fixed
	DestAddr Variable
}

// Len implements the Data interface.
func (ds *DestSme) Len() int {
	return ds.Flag.Len() + ds.Ton.Len() + ds.Npi.Len() + ds.DestAddr.Len()
}

// Raw implements the Data interface.
func (ds *DestSme) Raw() interface{} {
	return ds.Bytes()
}

// String implements the Data interface.
func (ds *DestSme) String() string {
	return ds.Flag.String() + "," + ds.Ton.String() + "," + ds.Npi.String() + "," + ds.DestAddr.String()
}

// Bytes implements the Data interface.
func (ds *DestSme) Bytes() []byte {
	var ret []byte
	ret = append(ret, ds.Flag.Bytes()...)
	ret = append(ret, ds.Ton.Bytes()...)
	ret = append(ret, ds.Npi.Bytes()...)
	ret = append(ret, ds.DestAddr.Bytes()...)
	return ret
}

// SerializeTo implements the Data interface.
func (ds *DestSme) SerializeTo(w io.Writer) error {
	_, err := w.Write(ds.Bytes())
	return err
}

// DestSmeList contains a list of DestSme
type DestSmeList struct {
	Data []DestSme
}

// Len implements the Data interface.
func (dsl *DestSmeList) Len() int {
	var ret int
	for i := range dsl.Data {
		ret = ret + dsl.Data[i].Len()
	}
	return ret
}

// Raw implements the Data interface.
func (dsl *DestSmeList) Raw() interface{} {
	return dsl.Bytes()
}

// String implements the Data interface.
func (dsl *DestSmeList) String() string {
	var ret string
	for i := range dsl.Data {
		ret = ret + dsl.Data[i].String() + ";"
	}
	return ret
}

// Bytes implements the Data interface.
func (dsl *DestSmeList) Bytes() []byte {
	var ret []byte
	for i := range dsl.Data {
		ret = append(ret, dsl.Data[i].Bytes()...)
	}
	return ret
}

// SerializeTo implements the Data interface.
func (dsl *DestSmeList) SerializeTo(w io.Writer) error {
	_, err := w.Write(dsl.Bytes())
	return err
}

// UnSme is a PDU field used for unsuccess sme addreses.
type UnSme struct {
	Ton      Fixed
	Npi      Fixed
	DestAddr Variable
	ErrCode  Variable
}

// Len implements the Data interface.
func (us *UnSme) Len() int {
	return us.Ton.Len() + us.Npi.Len() + us.DestAddr.Len() + us.ErrCode.Len()
}

// Raw implements the Data interface.
func (us *UnSme) Raw() interface{} {
	return us.Bytes()
}

// String implements the Data interface.
func (us *UnSme) String() string {
	return us.Ton.String() + "," + us.Npi.String() + "," + us.DestAddr.String() + "," + strconv.Itoa(int(binary.BigEndian.Uint32(us.ErrCode.Data)))
}

// Bytes implements the Data interface.
func (us *UnSme) Bytes() []byte {
	var ret []byte
	ret = append(ret, us.Ton.Bytes()...)
	ret = append(ret, us.Npi.Bytes()...)
	ret = append(ret, us.DestAddr.Bytes()...)
	ret = append(ret, us.ErrCode.Bytes()...)
	return ret
}

// SerializeTo implements the Data interface.
func (us *UnSme) SerializeTo(w io.Writer) error {
	_, err := w.Write(us.Bytes())
	return err
}

// UnSmeList contains a list of UnSme
type UnSmeList struct {
	Data []UnSme
}

// Len implements the Data interface.
func (usl *UnSmeList) Len() int {
	var ret int
	for i := range usl.Data {
		ret = ret + usl.Data[i].Len()
	}
	return ret
}

// Raw implements the Data interface.
func (usl *UnSmeList) Raw() interface{} {
	return usl.Bytes()
}

// String implements the Data interface.
func (usl *UnSmeList) String() string {
	var ret string
	for i := range usl.Data {
		ret = ret + usl.Data[i].String() + ";"
	}
	return ret
}

// Bytes implements the Data interface.
func (usl *UnSmeList) Bytes() []byte {
	var ret []byte
	for i := range usl.Data {
		ret = append(ret, usl.Data[i].Bytes()...)
	}
	return ret
}

// SerializeTo implements the Data interface.
func (usl *UnSmeList) SerializeTo(w io.Writer) error {
	_, err := w.Write(usl.Bytes())
	return err
}

// UDH is a PDU field used for user data header.
type UDH struct {
	IEI      Fixed
	IELength Fixed
	IEData   Variable
}

// Len implements the Data interface.
func (udh *UDH) Len() int {
	return udh.IEI.Len() + udh.IELength.Len() + udh.IEData.Len()
}

// Raw implements the Data interface.
func (udh *UDH) Raw() interface{} {
	return udh.Bytes()
}

// String implements the Data interface.
func (udh *UDH) String() string {
	return udh.IEI.String() + "," + udh.IELength.String() + "," + udh.IEData.String()
}

// Bytes implements the Data interface.
func (udh *UDH) Bytes() []byte {
	var ret []byte
	ret = append(ret, udh.IEI.Bytes()...)
	ret = append(ret, udh.IELength.Bytes()...)
	ret = append(ret, udh.IEData.Bytes()...)
	return ret
}

// SerializeTo implements the Data interface.
func (udh *UDH) SerializeTo(w io.Writer) error {
	_, err := w.Write(udh.Bytes())
	return err
}

// UDHList contains a list of UDH.
type UDHList struct {
	Data []UDH
}

// Len implements the Data interface.
func (udhl *UDHList) Len() int {
	var ret int
	for i := range udhl.Data {
		ret = ret + udhl.Data[i].Len()
	}
	return ret
}

// Raw implements the Data interface.
func (udhl *UDHList) Raw() interface{} {
	return udhl.Bytes()
}

// String implements the Data interface.
func (udhl *UDHList) String() string {
	var ret string
	for i := range udhl.Data {
		ret = ret + udhl.Data[i].String() + ";"
	}
	return ret
}

// Bytes implements the Data interface.
func (udhl *UDHList) Bytes() []byte {
	var ret []byte
	for i := range udhl.Data {
		ret = append(ret, udhl.Data[i].Bytes()...)
	}
	return ret
}

// SerializeTo implements the Data interface.
func (udhl *UDHList) SerializeTo(w io.Writer) error {
	_, err := w.Write(udhl.Bytes())
	return err
}
