// Copyright 2015 go-smpp authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package smpp

import (
	"crypto/tls"
	"fmt"
	"math/rand"
	"time"

	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
)

// Transceiver implements an SMPP transceiver.
//
// The API is a combination of the Transmitter and Receiver.
type Transceiver struct {
	Addr               string        // Server address in form of host:port.
	User               string        // Username.
	Passwd             string        // Password.
	SystemType         string        // System type, default empty.
	EnquireLink        time.Duration // Enquire link interval, default 10s.
	EnquireLinkTimeout time.Duration // Time after last EnquireLink response when connection considered down
	RespTimeout        time.Duration // Response timeout, default 1s.
	BindInterval       time.Duration // Binding retry interval
	TLS                *tls.Config   // TLS client settings, optional.
	Handler            HandlerFunc   // Receiver handler, optional.
	RateLimiter        RateLimiter   // Rate limiter, optional.
	WindowSize         uint

	Transmitter
}

// Bind implements the ClientConn interface.
func (t *Transceiver) Bind() <-chan ConnStatus {
	t.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	t.cl.Lock()
	defer t.cl.Unlock()
	if t.cl.client != nil {
		return t.cl.Status
	}
	t.tx.Lock()
	t.tx.inflight = make(map[uint32]chan *tx)
	t.tx.Unlock()
	c := &client{
		Addr:               t.Addr,
		TLS:                t.TLS,
		Status:             make(chan ConnStatus, 1),
		BindFunc:           t.bindFunc,
		EnquireLink:        t.EnquireLink,
		EnquireLinkTimeout: t.EnquireLinkTimeout,
		RespTimeout:        t.RespTimeout,
		WindowSize:         t.WindowSize,
		RateLimiter:        t.RateLimiter,
		BindInterval:       t.BindInterval,
	}
	t.cl.client = c
	c.init()
	go c.Bind()
	return c.Status
}

func (t *Transceiver) bindFunc(c Conn) error {
	p := pdu.NewBindTransceiver()
	f := p.Fields()
	f.Set(pdufield.SystemID, t.User)
	f.Set(pdufield.Password, t.Passwd)
	f.Set(pdufield.SystemType, t.SystemType)
	resp, err := bind(c, p)
	if err != nil {
		return err
	}
	if resp.Header().ID != pdu.BindTransceiverRespID {
		return fmt.Errorf("unexpected response for BindTransceiver: %s",
			resp.Header().ID)
	}
	go t.handlePDU(t.Handler)
	return nil
}
