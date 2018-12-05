//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux/coap"
	log "github.com/mainflux/mainflux/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mux "github.com/dereulenspiegel/coap-mux"
	gocoap "github.com/dustin/go-coap"
	"github.com/mainflux/mainflux"
)

const protocol = "coap"

var (
	errBadRequest = errors.New("bad request")
	errBadOption  = errors.New("bad option")
	auth          mainflux.ThingsServiceClient
	logger        log.Logger
	pingPeriod    time.Duration
)

type handler func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message

func notFoundHandler(l *net.UDPConn, a *net.UDPAddr, m *gocoap.Message) *gocoap.Message {
	if m.IsConfirmable() {
		return &gocoap.Message{
			Type: gocoap.Acknowledgement,
			Code: gocoap.NotFound,
		}
	}
	return nil
}

//MakeHTTPHandler creates handler for version endpoint.
func MakeHTTPHandler() http.Handler {
	b := bone.New()
	b.GetFunc("/version", mainflux.Version(protocol))
	b.Handle("/metrics", promhttp.Handler())

	return b
}

// MakeCOAPHandler creates handler for CoAP messages.
func MakeCOAPHandler(svc coap.Service, tc mainflux.ThingsServiceClient, l log.Logger, responses chan<- string, pp time.Duration) gocoap.Handler {
	auth = tc
	logger = l
	pingPeriod = pp
	r := mux.NewRouter()
	r.Handle("/channels/{id}/messages", gocoap.FuncHandler(receive(svc))).Methods(gocoap.POST)
	r.Handle("/channels/{id}/messages", gocoap.FuncHandler(observe(svc, responses)))
	r.NotFoundHandler = gocoap.FuncHandler(notFoundHandler)

	return r
}

func authorize(msg *gocoap.Message, res *gocoap.Message, cid string) (string, error) {
	// Device Key is passed as Uri-Query parameter, which option ID is 15 (0xf).
	key, err := authKey(msg.Option(gocoap.URIQuery))
	if err != nil {
		switch err {
		case errBadOption:
			res.Code = gocoap.BadOption
		case errBadRequest:
			res.Code = gocoap.BadRequest
		}

		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := auth.CanAccess(ctx, &mainflux.AccessReq{Token: key, ChanID: cid})

	if err != nil {
		e, ok := status.FromError(err)
		if ok {
			switch e.Code() {
			case codes.PermissionDenied:
				res.Code = gocoap.Forbidden
			default:
				res.Code = gocoap.ServiceUnavailable
			}
			return "", err
		}
		res.Code = gocoap.InternalServerError
	}
	return id.GetValue(), nil
}

func receive(svc coap.Service) handler {
	return func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message {
		// By default message is NonConfirmable, so
		// NonConfirmable response is sent back.
		res := &gocoap.Message{
			Type: gocoap.NonConfirmable,
			// According to https://tools.ietf.org/html/rfc7252#page-47: If the POST
			// succeeds but does not result in a new resource being created on the
			// server, the response SHOULD have a 2.04 (Changed) Response Code.
			Code:      gocoap.Changed,
			MessageID: msg.MessageID,
			Token:     msg.Token,
			Payload:   []byte{},
		}

		if msg.IsConfirmable() {
			res.Type = gocoap.Acknowledgement
			res.SetOption(gocoap.ContentFormat, gocoap.AppJSON)
			if len(msg.Payload) == 0 {
				res.Code = gocoap.BadRequest
				return res
			}
		}

		chanID := mux.Var(msg, "id")
		if chanID == "" {
			res.Code = gocoap.NotFound
			return res
		}

		publisher, err := authorize(msg, res, chanID)
		if err != nil {
			res.Code = gocoap.Forbidden
			return res
		}

		rawMsg := mainflux.RawMessage{
			Channel:   chanID,
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

func observe(svc coap.Service, responses chan<- string) handler {
	return func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message {
		res := &gocoap.Message{
			Type:      gocoap.Acknowledgement,
			Code:      gocoap.Content,
			MessageID: msg.MessageID,
			Token:     msg.Token,
			Payload:   []byte{},
		}
		res.SetOption(gocoap.ContentFormat, gocoap.AppJSON)

		chanID := mux.Var(msg, "id")
		if chanID == "" {
			res.Code = gocoap.NotFound
			return res
		}

		publisher, err := authorize(msg, res, chanID)
		if err != nil {
			res.Code = gocoap.Forbidden
			logger.Warn(fmt.Sprintf("Failed to authorize: %s", err))
			return res
		}

		obsID := fmt.Sprintf("%x-%s-%s", msg.Token, publisher, chanID)

		if msg.Type == gocoap.Acknowledgement {
			responses <- obsID
			return nil
		}

		if value, ok := msg.Option(gocoap.Observe).(uint32); (ok && value == 1) || msg.Type == gocoap.Reset {
			svc.Unsubscribe(obsID)
		}

		if value, ok := msg.Option(gocoap.Observe).(uint32); ok && value == 0 {
			res.AddOption(gocoap.Observe, 1)
			o := coap.NewObserver()
			if err := svc.Subscribe(chanID, obsID, o); err != nil {
				logger.Warn(fmt.Sprintf("Failed to subscribe to NATS subject: %s", err))
				res.Code = gocoap.InternalServerError
				return res
			}

			go handleMessage(conn, addr, o, msg)
			go ping(svc, obsID, conn, addr, o, msg)
			go cancel(o)
		}

		return res
	}
}

func cancel(observer *coap.Observer) {
	<-observer.Cancel
	close(observer.Messages)
	observer.StoreExpired(true)
}

func handleMessage(conn *net.UDPConn, addr *net.UDPAddr, o *coap.Observer, msg *gocoap.Message) {
	notifyMsg := *msg
	notifyMsg.Type = gocoap.NonConfirmable
	notifyMsg.Code = gocoap.Content
	notifyMsg.RemoveOption(gocoap.URIQuery)
	for {
		msg, ok := <-o.Messages
		if !ok {
			return
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to parse received message: %s", err))
			continue
		}

		notifyMsg.Payload = payload
		notifyMsg.MessageID = o.LoadMessageID()
		buff := new(bytes.Buffer)
		observe := uint64(notifyMsg.MessageID)
		if err := binary.Write(buff, binary.BigEndian, observe); err != nil {
			logger.Warn(fmt.Sprintf("Failed to generate Observe option value: %s", err))
			continue
		}

		observeVal := buff.Bytes()
		notifyMsg.SetOption(gocoap.Observe, observeVal[len(observeVal)-3:])

		if err := gocoap.Transmit(conn, addr, notifyMsg); err != nil {
			logger.Warn(fmt.Sprintf("Failed to send message to observer: %s", err))
		}
	}
}

func ping(svc coap.Service, obsID string, conn *net.UDPConn, addr *net.UDPAddr, o *coap.Observer, msg *gocoap.Message) {
	pingMsg := *msg
	pingMsg.Payload = []byte{}
	pingMsg.Type = gocoap.Confirmable
	pingMsg.RemoveOption(gocoap.URIQuery)
	// According to RFC (https://tools.ietf.org/html/rfc7641#page-18), CON message must be sent at least every
	// 24 hours. Deafault value of pingPeriod is 12.
	t := time.NewTicker(pingPeriod * time.Hour)
	defer t.Stop()
	for {
		select {
		case _, ok := <-t.C:
			if !ok || o.LoadExpired() {
				return
			}

			o.StoreExpired(true)
			timeout := float64(coap.AckTimeout)
			logger.Info(fmt.Sprintf("Ping client %s.", obsID))
			for i := 0; i < coap.MaxRetransmit; i++ {
				pingMsg.MessageID = o.LoadMessageID()
				gocoap.Transmit(conn, addr, pingMsg)
				time.Sleep(time.Duration(timeout * coap.AckRandomFactor))
				if !o.LoadExpired() {
					break
				}
				timeout = 2 * timeout
			}

			if o.LoadExpired() {
				svc.Unsubscribe(obsID)
				return
			}
		case <-o.Cancel:
			return
		}
	}
}
