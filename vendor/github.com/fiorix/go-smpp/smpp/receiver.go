// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package smpp

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
)

// Receiver implements an SMPP client receiver.
type Receiver struct {
	Addr                 string
	User                 string
	Passwd               string
	SystemType           string
	EnquireLink          time.Duration
	EnquireLinkTimeout   time.Duration // Time after last EnquireLink response when connection considered down
	BindInterval         time.Duration // Binding retry interval
	MergeInterval        time.Duration // Time in which Receiver waits for the parts of the long messages
	MergeCleanupInterval time.Duration // How often to cleanup expired message parts
	TLS                  *tls.Config
	Handler              HandlerFunc
	SkipAutoRespondIDs   []pdu.ID

	chanClose chan struct{}

	// struct which holds the map of MergeHolders for the merging of the long incoming messages.
	// It is used only if the incoming PDU holds UDH data and Receiver has MergeInterval > 0.
	mg struct {
		mergeHolders map[int]*MergeHolder
		sync.Mutex
	}

	cl struct {
		*client
		sync.Mutex
	}
}

// HandlerFunc is the handler function that a Receiver calls
// when a new PDU arrives.
type HandlerFunc func(p pdu.Body)

// MergeHolder is a struct which holds the slice of MessageParts for the merging of a long incoming message.
type MergeHolder struct {
	MessageID     int
	MessageParts  []*MessagePart // Slice with the parts of the message
	PartsCount    int
	LastWriteTime time.Time
}

// MessagePart is a struct which holds the data of the part of a long incoming message.
type MessagePart struct {
	PartID int
	Data   *bytes.Buffer
}

// Bind starts the Receiver. It creates a persistent connection
// to the server, update its status via the returned channel,
// and calls the registered Handler when new PDU arrives.
//
// Bind implements the ClientConn interface.
func (r *Receiver) Bind() <-chan ConnStatus {
	r.cl.Lock()
	defer r.cl.Unlock()

	r.chanClose = make(chan struct{})

	if r.cl.client != nil {
		return r.cl.Status
	}

	c := &client{
		Addr:               r.Addr,
		TLS:                r.TLS,
		EnquireLink:        r.EnquireLink,
		EnquireLinkTimeout: r.EnquireLinkTimeout,
		Status:             make(chan ConnStatus, 1),
		BindFunc:           r.bindFunc,
		BindInterval:       r.BindInterval,
	}
	r.cl.client = c

	c.init()
	go c.Bind()

	// Set up message merging if requested
	if r.MergeInterval > 0 {
		if r.MergeCleanupInterval == 0 {
			r.MergeCleanupInterval = 1 * time.Second
		}

		r.mg.mergeHolders = make(map[int]*MergeHolder)
		go r.mergeCleaner()
	}

	return c.Status
}

func (r *Receiver) bindFunc(c Conn) error {
	p := pdu.NewBindReceiver()
	f := p.Fields()
	f.Set(pdufield.SystemID, r.User)
	f.Set(pdufield.Password, r.Passwd)
	f.Set(pdufield.SystemType, r.SystemType)
	resp, err := bind(c, p)
	if err != nil {
		return err
	}
	if resp.Header().ID != pdu.BindReceiverRespID {
		return fmt.Errorf("unexpected response for BindReceiver: %s",
			resp.Header().ID)
	}

	// Clean the map in case of rebind, because message id numbering resets after reconnection
	// and older IDs are no longer valid
	if r.MergeInterval > 0 {
		r.mg.Lock()
		r.mg.mergeHolders = make(map[int]*MergeHolder)
		r.mg.Unlock()
	}

	if r.Handler != nil {
		go r.handlePDU()
	}

	return nil
}

func idInList(id pdu.ID, list []pdu.ID) bool {
	for _, x := range list {
		if x == id {
			return true
		}
	}
	return false
}

func (r *Receiver) handlePDU() {
	var (
		ok                bool
		sm                *pdufield.SM
		udhList           *pdufield.UDHList
		msgID, partsCount int
		mh                *MergeHolder
		orderedBodies     []*bytes.Buffer
	)
	autoRespondDeliver := !idInList(pdu.DeliverSMID, r.SkipAutoRespondIDs)

loop:
	for {
		p, err := r.cl.Read()
		if err != nil || p == nil {
			break
		}

		if p.Header().ID == pdu.DeliverSMID && autoRespondDeliver { // Send DeliverSMResp
			pResp := pdu.NewDeliverSMRespSeq(p.Header().Seq)
			r.cl.Write(pResp)
		}

		if r.MergeInterval == 0 { // Handle the PDU if merging is not needed
			r.Handler(p)
			continue
		}

		sm, ok = p.Fields()[pdufield.ShortMessage].(*pdufield.SM)
		if !ok {
			// PDU is malformed, do not process
			continue
		}

		udhList, ok = p.Fields()[pdufield.GSMUserData].(*pdufield.UDHList)
		if !ok { // Check if GSMUserData is present inside the PDU, do not try to merge if it's not
			r.Handler(p)
			continue
		}

		for _, udh := range udhList.Data {
			switch udh.IEI.Data {
			case 0x00: // Concatenated short messages, 8-bit reference number
				if int(udh.IELength.Data) != 3 { // Contains message ID, parts count and part number
					// PDU is malformed, do not process
					break
				}

				// Get message ID and total count of its parts
				msgID = int(udh.IEData.Data[0])
				partsCount = int(udh.IEData.Data[1])

				// Check if message part was already added to a MergeHolder
				r.mg.Lock()
				if mh, ok = r.mg.mergeHolders[msgID]; !ok {
					mh = &MergeHolder{
						MessageID:  msgID,
						PartsCount: partsCount,
					}

					r.mg.mergeHolders[msgID] = mh
				}
				r.mg.Unlock()

				// Add current part of the message to the slice
				mh.MessageParts = append(mh.MessageParts, &MessagePart{
					PartID: int(udh.IEData.Data[2]),
					Data:   bytes.NewBuffer(sm.Data),
				})
				mh.LastWriteTime = time.Now()

				// Check if we have all the parts of the message
				if len(mh.MessageParts) != mh.PartsCount {
					continue loop
				}

				// Order up PDUs
				orderedBodies = make([]*bytes.Buffer, partsCount)
				for _, mp := range mh.MessageParts {
					orderedBodies[mp.PartID-1] = mp.Data
				}

				// Merge PDUs
				var buf bytes.Buffer
				for _, body := range orderedBodies {
					buf.Write(body.Bytes())
				}

				p.Fields().Set(pdufield.ShortMessage, buf.Bytes())

				// Handle
				r.Handler(p)
			}
		}
	}
}

func (r *Receiver) mergeCleaner() {
	timer := time.NewTimer(r.MergeCleanupInterval)

	for {
		select {
		case <-timer.C:
			r.mg.Lock()
			for _, mHolder := range r.mg.mergeHolders {
				if time.Since(mHolder.LastWriteTime) > r.MergeInterval { // Message has expired, remove
					delete(r.mg.mergeHolders, mHolder.MessageID)
				}
			}
			r.mg.Unlock()

		case <-r.chanClose:
			return
		}
	}
}

// Close implements the ClientConn interface.
func (r *Receiver) Close() error {
	r.cl.Lock()
	defer r.cl.Unlock()
	if r.cl.client == nil {
		return ErrNotConnected
	}
	close(r.chanClose)
	return r.cl.Close()
}
