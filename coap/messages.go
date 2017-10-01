package adapter

import (
	"bytes"
	"log"
	"net"

	mux "github.com/dereulenspiegel/coap-mux"
	coap "github.com/dustin/go-coap"
	"github.com/mainflux/mainflux/writer"
)

// sendMessage sends the message to NATS
func (ca *CoAPAdapter) sendMessage(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	log.Printf("Got message in sendMessage: path=%q: %#v from %v", m.Path(), m, a)
	var res *coap.Message
	if m.IsConfirmable() {
		res = &coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Content,
			MessageID: m.MessageID,
			Token:     m.Token,
			Payload:   []byte(""),
		}
		res.SetOption(coap.ContentFormat, coap.AppJSON)
	}

	if len(m.Payload) == 0 {
		if m.IsConfirmable() {
			res.Payload = []byte("{\"res\": \"Error: msg len can not be 0\"}")
		}
		return res
	}

	// Channel ID
	cid := mux.Var(m, "channel_id")

	// Publish message via NATS
	n := writer.RawMessage{}
	n.Channel = cid
	n.Publisher = ""
	n.Protocol = "coap"
	n.Payload = m.Payload

	if err := ca.repo.Save(n); err != nil {
		if m.IsConfirmable() {
			res.Code = coap.InternalServerError
		}
		return res
	}

	if m.IsConfirmable() {
		res.Code = coap.Changed
	}
	return res
}

// registerObserver functions adds observer struct to the observers map
func (ca *CoAPAdapter) registerObserver(o Observer, cid string) {
	found := false
	for _, v := range ca.obsMap[cid] {
		if v.addr == o.addr && bytes.Compare(v.message.Token, o.message.Token) == 0 {
			found = true
			break
		}
	}
	if !found {
		log.Println("Register " + cid)
		log.Printf("o.message = %v", o.message)
		ca.obsMap[cid] = append(ca.obsMap[cid], o)
	}
}

// deregisterObserver functions removes observer struct from the observers map
func (ca *CoAPAdapter) deregisterObserver(o Observer, cid string) {
	for k, v := range ca.obsMap[cid] {
		if bytes.Compare(v.message.Token, o.message.Token) == 0 {
			// Observer found, remove it from array
			log.Println("Deregister " + cid)
			ca.obsMap[cid] = append((ca.obsMap[cid])[:k], (ca.obsMap[cid])[k+1:]...)
		}
	}
}

// observeMessage adds client to the observers map
func (ca *CoAPAdapter) observeMessage(l *net.UDPConn, a *net.UDPAddr, m *coap.Message) *coap.Message {
	log.Printf("Got message in observeMessage: path=%q: %#v from %v", m.Path(), m, a)
	var res *coap.Message

	if m.IsConfirmable() {
		res = &coap.Message{
			Type:      coap.Acknowledgement,
			Code:      coap.Content,
			MessageID: m.MessageID,
			Token:     m.Token,
			Payload:   []byte(""),
		}
		res.SetOption(coap.ContentFormat, coap.AppJSON)
	}

	// Channel ID
	cid := mux.Var(m, "channel_id")

	// Observer
	o := Observer{
		conn:    l,
		addr:    a,
		message: m,
	}

	if m.Option(coap.Observe) == nil {
		if m.IsConfirmable() {
			res.Code = coap.BadRequest
		}
		return res
	}

	if value, ok := m.Option(coap.Observe).(uint32); ok {
		if value == 0 {
			ca.registerObserver(o, cid)
		} else {
			ca.deregisterObserver(o, cid)
		}
	} else {
		// Interop - old deregister was when there was no Observe option provided
		ca.deregisterObserver(o, cid)
	}

	if m.IsConfirmable() {
		res.Code = coap.Valid
	}
	return res
}

// obsTransmit transmits the message to observing clients
func (ca *CoAPAdapter) obsTransmit(n writer.RawMessage) {
	for _, v := range ca.obsMap[n.Channel] {
		msg := *(v.message)
		msg.Payload = n.Payload

		log.Printf("ca.obsMap[cid] = %v", v)
		log.Printf("msg = %v", msg)

		msg.SetOption(coap.ContentFormat, coap.AppJSON)
		msg.SetOption(coap.LocationPath, msg.Path())

		log.Printf("Transmitting %v", msg)
		err := coap.Transmit(v.conn, v.addr, msg)
		if err != nil {
			log.Printf("Error on transmitter, stopping: %v", err)
			return
		}
	}

}
