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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	gocoap "github.com/dustin/go-coap"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	log "github.com/mainflux/mainflux/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	protocol                   = "coap"
	senMLJSON gocoap.MediaType = 110
	senMLCBOR gocoap.MediaType = 112
)

var (
	errBadRequest        = errors.New("bad request")
	errBadOption         = errors.New("bad option")
	errMalformedSubtopic = errors.New("malformed subtopic")
	channelRegExp        = regexp.MustCompile(`^/?channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)
)

var (
	auth       mainflux.ThingsServiceClient
	logger     log.Logger
	pingPeriod time.Duration
)

type handler func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message

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
	return mux(svc, responses)
}

func mux(svc coap.Service, responses chan<- string) gocoap.Handler {
	return gocoap.FuncHandler(func(conn *net.UDPConn, addr *net.UDPAddr, msg *gocoap.Message) *gocoap.Message {
		path := msg.PathString()
		if !channelRegExp.Match([]byte(path)) {
			logger.Info(fmt.Sprintf("path %s not found", path))
			return &gocoap.Message{
				Type:      gocoap.NonConfirmable,
				Code:      gocoap.NotFound,
				MessageID: msg.MessageID,
				Token:     msg.Token,
			}
		}
		// Allow "/" to be a part of the path.
		if strings.HasPrefix(path, "/") {
			msg.SetPathString(path[1:])
		}
		switch msg.Code {
		case gocoap.GET:
			return observe(svc, responses)(conn, addr, msg)
		default:
			return receive(svc, msg)
		}
	})
}

func id(msg *gocoap.Message) string {
	vars := strings.Split(msg.PathString(), "/")
	if len(vars) > 1 {
		return vars[1]
	}
	return ""
}

func subtopic(msg *gocoap.Message) string {
	path := msg.PathString()
	pos := 0
	for i, c := range path {
		if c == '/' {
			pos++
		}
		if pos == 3 {
			return path[i:]
		}
	}
	return ""
}

func authorize(msg *gocoap.Message, res *gocoap.Message, cid string) (string, error) {
	// Device Key is passed as Uri-Query parameter, which option ID is 15 (0xf).
	query := msg.Option(gocoap.URIQuery)
	queryStr, ok := query.(string)
	if !ok {
		res.Code = gocoap.BadRequest
		return "", errBadRequest
	}

	params, err := url.ParseQuery(queryStr)
	if err != nil {
		res.Code = gocoap.BadRequest
		return "", errBadRequest
	}

	auths, ok := params["authorization"]
	if !ok || len(auths) != 1 {
		res.Code = gocoap.BadRequest
		return "", errBadRequest
	}

	key := auths[0]

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

func fmtSubtopic(msg *gocoap.Message) (string, error) {
	subtopic := subtopic(msg)
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic = strings.Replace(subtopic, "/", ".", -1)

	elems := strings.Split(subtopic, ".")
	filteredElems := []string{}
	for _, elem := range elems {
		if elem == "" {
			continue
		}

		if len(elem) > 1 && (strings.Contains(elem, "*") || strings.Contains(elem, ">")) {
			return "", errMalformedSubtopic
		}

		filteredElems = append(filteredElems, elem)
	}

	subtopic = strings.Join(filteredElems, ".")

	return subtopic, nil
}

func receive(svc coap.Service, msg *gocoap.Message) *gocoap.Message {
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

	chanID := id(msg)
	if chanID == "" {
		res.Code = gocoap.NotFound
		return res
	}

	subtopic, err := fmtSubtopic(msg)
	if err != nil {
		res.Code = gocoap.BadRequest
		return res
	}

	ct, err := contentType(msg)
	if err != nil {
		ct = ""
	}

	publisher, err := authorize(msg, res, chanID)
	if err != nil {
		res.Code = gocoap.Forbidden
		return res
	}

	rawMsg := mainflux.RawMessage{
		Channel:     chanID,
		Subtopic:    subtopic,
		Publisher:   publisher,
		ContentType: ct,
		Protocol:    protocol,
		Payload:     msg.Payload,
	}

	if err := svc.Publish(context.Background(), "", rawMsg); err != nil {
		res.Code = gocoap.InternalServerError
	}

	return res
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

		chanID := id(msg)
		if chanID == "" {
			res.Code = gocoap.NotFound
			return res
		}

		subtopic, err := fmtSubtopic(msg)
		if err != nil {
			res.Code = gocoap.BadRequest
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
			if err := svc.Subscribe(chanID, subtopic, obsID, o); err != nil {
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

		notifyMsg.Payload = msg.Payload
		notifyMsg.MessageID = o.LoadMessageID()
		buff := new(bytes.Buffer)
		observe := uint64(notifyMsg.MessageID)
		if err := binary.Write(buff, binary.BigEndian, observe); err != nil {
			logger.Warn(fmt.Sprintf("Failed to generate Observe option value: %s", err))
			continue
		}

		observeVal := buff.Bytes()
		notifyMsg.SetOption(gocoap.Observe, observeVal[len(observeVal)-3:])

		coapCT := senMLJSON
		switch msg.ContentType {
		case mainflux.SenMLJSON:
			coapCT = senMLJSON
		case mainflux.SenMLCBOR:
			coapCT = senMLCBOR
		}
		notifyMsg.SetOption(gocoap.ContentFormat, coapCT)

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

func contentType(msg *gocoap.Message) (string, error) {
	ctid, ok := msg.Option(gocoap.ContentFormat).(gocoap.MediaType)
	if !ok {
		return "", errBadRequest
	}

	ct := ""
	switch ctid {
	case senMLJSON:
		ct = mainflux.SenMLJSON
	case senMLCBOR:
		ct = mainflux.SenMLCBOR
	}

	return ct, nil
}
