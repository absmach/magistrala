//
// Copyright (c) 2018
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/ws"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const protocol = "ws"

var (
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errMalformedData      = errors.New("malformed request data")
	upgrader              = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	auth   mainflux.ThingsServiceClient
	logger log.Logger
)

// MakeHandler returns http handler with handshake endpoint.
func MakeHandler(svc ws.Service, tc mainflux.ThingsServiceClient, l log.Logger) http.Handler {
	auth = tc
	logger = l

	mux := bone.New()
	mux.GetFunc("/channels/:id/messages", handshake(svc))
	mux.GetFunc("/version", mainflux.Version("websocket"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func handshake(svc ws.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sub, err := authorize(r)
		if err != nil {
			switch err {
			case errMalformedData:
				logger.Warn(fmt.Sprintf("Empty channel id"))
				w.WriteHeader(http.StatusBadRequest)
				return
			case things.ErrUnauthorizedAccess:
				w.WriteHeader(http.StatusForbidden)
				return
			default:
				logger.Warn(fmt.Sprintf("Failed to authorize: %s", err))
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		// Create new ws connection.
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to upgrade connection to websocket: %s", err))
			return
		}
		sub.conn = conn

		sub.channel = ws.NewChannel()
		if err := svc.Subscribe(sub.chanID, sub.channel); err != nil {
			logger.Warn(fmt.Sprintf("Failed to subscribe to NATS subject: %s", err))
			conn.Close()
			return
		}
		go sub.listen()

		// Start listening for messages from NATS.
		go sub.broadcast(svc)
	}
}

func authorize(r *http.Request) (subscription, error) {
	authKey := r.Header.Get("Authorization")
	if authKey == "" {
		authKeys := bone.GetQuery(r, "authorization")
		if len(authKeys) == 0 {
			return subscription{}, things.ErrUnauthorizedAccess
		}
		authKey = authKeys[0]
	}

	// Extract ID from /channels/:id/messages.
	chanID := bone.GetValue(r, "id")
	if chanID == "" {
		return subscription{}, errMalformedData
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := auth.CanAccess(ctx, &mainflux.AccessReq{Token: authKey, ChanID: chanID})
	if err != nil {
		e, ok := status.FromError(err)
		if ok && e.Code() == codes.PermissionDenied {
			return subscription{}, things.ErrUnauthorizedAccess
		}
		return subscription{}, err
	}

	sub := subscription{
		pubID:  id.GetValue(),
		chanID: chanID,
	}

	return sub, nil
}

type subscription struct {
	pubID   string
	chanID  string
	conn    *websocket.Conn
	channel *ws.Channel
}

func (sub subscription) broadcast(svc ws.Service) {
	for {
		_, payload, err := sub.conn.ReadMessage()
		if websocket.IsUnexpectedCloseError(err) {
			sub.channel.Close()
			return
		}
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to read message: %s", err))
			return
		}
		msg := mainflux.RawMessage{
			Channel:   sub.chanID,
			Publisher: sub.pubID,
			Protocol:  protocol,
			Payload:   payload,
		}
		if err := svc.Publish(msg); err != nil {
			logger.Warn(fmt.Sprintf("Failed to publish message to NATS: %s", err))
			if err == ws.ErrFailedConnection {
				sub.conn.Close()
				sub.channel.Closed <- true
				return
			}
		}
	}
}

func (sub subscription) listen() {
	for msg := range sub.channel.Messages {
		if err := sub.conn.WriteMessage(websocket.TextMessage, msg.Payload); err != nil {
			logger.Warn(fmt.Sprintf("Failed to broadcast message to thing: %s", err))
		}
	}
}
