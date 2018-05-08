package api

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mainflux/mainflux/coap"
	"github.com/mainflux/mainflux/coap/nats"
	manager "github.com/mainflux/mainflux/manager/client"

	"math/rand"

	mux "github.com/dereulenspiegel/coap-mux"
	gocoap "github.com/dustin/go-coap"
	"github.com/mainflux/mainflux"
)

var (
	errBadRequest = errors.New("bad request")
	errBadOption  = errors.New("bad option")
	auth          manager.ManagerClient
)

const (
	maxPktLen = 1500
	network   = "udp"
	protocol  = "coap"
)

const (
	// Approximately number of supported requests per second
	timestamp = int64(time.Millisecond) * 31
)

type handler func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message

// NotFoundHandler handles erroneously formed requests.
func NotFoundHandler(l *net.UDPConn, a *net.UDPAddr, m *gocoap.Message) *gocoap.Message {
	if m.IsConfirmable() {
		return &gocoap.Message{
			Type: gocoap.Acknowledgement,
			Code: gocoap.NotFound,
		}
	}
	return nil
}

// MakeHandler function return new CoAP server with GET, POST and NOT_FOUND handlers.
func MakeHandler(svc coap.Service) gocoap.Handler {
	r := mux.NewRouter()
	r.Handle("/channels/{id}/messages", gocoap.FuncHandler(receive(svc))).Methods(gocoap.POST)
	r.Handle("/channels/{id}/messages", gocoap.FuncHandler(observe(svc))).Methods(gocoap.GET)
	r.NotFoundHandler = gocoap.FuncHandler(NotFoundHandler)
	return r
}

func receive(svc coap.Service) handler {
	return func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message {
		var res *gocoap.Message
		if msg.IsConfirmable() {
			res = &gocoap.Message{
				Type:      gocoap.Acknowledgement,
				Code:      gocoap.Content,
				MessageID: msg.MessageID,
				Token:     msg.Token,
				Payload:   []byte{},
			}
			res.SetOption(gocoap.ContentFormat, gocoap.AppJSON)
		}

		if len(msg.Payload) == 0 && msg.IsConfirmable() {
			res.Code = gocoap.BadRequest
			return res
		}

		cid := mux.Var(msg, "id")
		publisher, err := authorize(msg, res, cid)
		if err != nil {
			res.Code = gocoap.Unauthorized
			return res
		}

		rawMsg := mainflux.RawMessage{
			Channel:   cid,
			Publisher: publisher,
			Protocol:  protocol,
			Payload:   msg.Payload,
		}

		if err := svc.Publish(rawMsg); err != nil {
			res.Code = gocoap.InternalServerError
		}
		return res
	}
}

func observe(svc coap.Service) handler {
	return func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message {
		var res *gocoap.Message
		if msg.IsConfirmable() {
			res = &gocoap.Message{
				Type:      gocoap.Acknowledgement,
				Code:      gocoap.Content,
				MessageID: msg.MessageID,
				Token:     msg.Token,
				Payload:   []byte{},
			}
			res.SetOption(gocoap.ContentFormat, gocoap.AppJSON)
		}

		cid := mux.Var(msg, "id")
		publisher, err := authorize(msg, res, cid)

		if err != nil {
			res.Code = gocoap.Unauthorized
			return res
		}

		if value, ok := msg.Option(gocoap.Observe).(uint32); ok && value == 1 {
			id := fmt.Sprintf("%s-%x", publisher, msg.Token)
			svc.Unsubscribe(id)
		}

		if value, ok := msg.Option(gocoap.Observe).(uint32); ok && value == 0 {
			ch := nats.Channel{
				Messages: make(chan mainflux.RawMessage),
				Closed:   make(chan bool),
				Timer:    make(chan bool),
				Notify:   make(chan bool),
			}
			id := fmt.Sprintf("%s-%x", publisher, msg.Token)
			if err := svc.Subscribe(cid, id, ch); err != nil {
				res.Code = gocoap.InternalServerError
				return res
			}
			go handleSub(svc, id, conn, addr, msg, ch)
			res.AddOption(gocoap.Observe, 0)
		}
		return res
	}
}

func sendMessage(svc coap.Service, id string, conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) error {
	buff := new(bytes.Buffer)
	now := time.Now().UnixNano() / timestamp
	if err := binary.Write(buff, binary.BigEndian, now); err != nil {
		return err
	}
	observeVal := buff.Bytes()
	msg.SetOption(gocoap.Observe, observeVal[len(observeVal)-3:])
	if msg.IsConfirmable() {
		timer := time.NewTimer(time.Duration(coap.AckTimeout))
		ch, err := svc.SetTimeout(id, timer, coap.AckTimeout)
		if err != nil {
			return err
		}
		return sendConfirmable(conn, addr, msg, ch)
	}
	return gocoap.Transmit(conn, addr, *msg)
}

func sendConfirmable(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message, ch chan bool) error {
	msg.SetOption(gocoap.MaxRetransmit, coap.MaxRetransmit)
	// Try to transmit MAX_RETRANSMITION times; every attempt duplicates timeout between transmission.
	for i := 0; i < coap.MaxRetransmit; i++ {
		if err := gocoap.Transmit(conn, addr, *msg); err != nil {
			return err
		}
		state, ok := <-ch
		if !state || !ok {
			return nil
		}
	}
	return nil
}

func handleSub(svc coap.Service, id string, conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message, ch nats.Channel) {
	// According to RFC (https://tools.ietf.org/html/rfc7641#page-18), CON message must be sent at least every
	// 24 hours. Since 24 hours is too long for our purposes, we use 12.
	ticker := time.NewTicker(12 * time.Hour)
	res := &gocoap.Message{
		Type:      gocoap.NonConfirmable,
		Code:      gocoap.Content,
		MessageID: msg.MessageID,
		Token:     msg.Token,
		Payload:   []byte{},
	}
	res.SetOption(gocoap.ContentFormat, gocoap.AppJSON)
	res.SetOption(gocoap.LocationPath, msg.Path())

loop:
	for {
		select {
		case <-ticker.C:
			res.Type = gocoap.Confirmable
			rand.Seed(time.Now().UnixNano())
			if err := sendMessage(svc, id, conn, addr, res); err != nil {
				svc.Unsubscribe(id)
				break loop
			}
		case rawMsg, ok := <-ch.Messages:
			if !ok {
				break loop
			}
			res.Type = gocoap.NonConfirmable
			res.Payload = rawMsg.Payload
			if err := sendMessage(svc, id, conn, addr, res); err != nil {
				break loop
			}
		}
	}
	ticker.Stop()
}
