// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-zoo/bone"
	"github.com/gorilla/websocket"
	"github.com/mainflux/mainflux"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/transformers/senml"
	"github.com/mainflux/mainflux/ws"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	protocol = "ws"
)

var (
	errUnauthorizedAccess = errors.New("missing or invalid credentials provided")
	errMalformedSubtopic  = errors.New("malformed subtopic")
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	auth              mainflux.ThingsServiceClient
	logger            log.Logger
	channelPartRegExp = regexp.MustCompile(`^/channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)
)

var contentTypes = map[string]int{
	senml.JSON: websocket.TextMessage,
	senml.CBOR: websocket.BinaryMessage,
}

// MakeHandler returns http handler with handshake endpoint.
func MakeHandler(svc ws.Service, tc mainflux.ThingsServiceClient, l log.Logger) http.Handler {
	auth = tc
	logger = l

	mux := bone.New()
	mux.GetFunc("/channels/:id/messages", handshake(svc))
	mux.GetFunc("/channels/:id/messages/*", handshake(svc))
	mux.GetFunc("/version", mainflux.Version("websocket"))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func handshake(svc ws.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sub, err := authorize(r)
		if err != nil {
			switch err {
			case things.ErrUnauthorizedAccess:
				w.WriteHeader(http.StatusForbidden)
				return
			default:
				logger.Warn(fmt.Sprintf("Failed to authorize: %s", err.Error()))
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}

		ct := contentType(r)

		channelParts := channelPartRegExp.FindStringSubmatch(r.RequestURI)
		if len(channelParts) < 2 {
			logger.Warn("Empty channel id or malformed url")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		sub.subtopic, err = parseSubtopic(channelParts[2])
		if err != nil {
			logger.Warn("Empty channel id or malformed url")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Create new ws connection.
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to upgrade connection to websocket: %s", err))
			return
		}
		sub.conn = conn

		logger.Debug(fmt.Sprintf("Successfully upgraded communication to WS on channel %s", sub.chanID))

		sub.channel = ws.NewChannel()
		if err := svc.Subscribe(sub.chanID, sub.subtopic, sub.channel); err != nil {
			logger.Warn(fmt.Sprintf("Failed to subscribe to NATS subject: %s", err))
			conn.Close()
			return
		}

		logger.Debug(fmt.Sprintf("Successfully subscribed to NATS channel %s", sub.chanID))

		go sub.listen()

		// Start listening for messages from NATS.
		go sub.broadcast(svc, ct)
	}
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
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

func authorize(r *http.Request) (subscription, error) {
	authKey := r.Header.Get("Authorization")
	if authKey == "" {
		authKeys := bone.GetQuery(r, "authorization")
		if len(authKeys) == 0 {
			logger.Debug("Missing authorization key.")
			return subscription{}, things.ErrUnauthorizedAccess
		}
		authKey = authKeys[0]
	}

	chanID := bone.GetValue(r, "id")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	id, err := auth.CanAccessByKey(ctx, &mainflux.AccessByKeyReq{Token: authKey, ChanID: chanID})
	if err != nil {
		e, ok := status.FromError(err)
		if ok && e.Code() == codes.PermissionDenied {
			return subscription{}, things.ErrUnauthorizedAccess
		}
		return subscription{}, err
	}
	logger.Debug(fmt.Sprintf("Successfully authorized client %s on channel %s", id.GetValue(), chanID))

	sub := subscription{
		pubID:  id.GetValue(),
		chanID: chanID,
	}

	return sub, nil
}

func contentType(r *http.Request) string {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		ctvals := bone.GetQuery(r, "content-type")
		if len(ctvals) == 0 {
			return ""
		}
		ct = ctvals[0]
	}

	return ct
}

type subscription struct {
	pubID    string
	chanID   string
	subtopic string
	conn     *websocket.Conn
	channel  *ws.Channel
}

func (sub subscription) broadcast(svc ws.Service, contentType string) {
	for {
		_, payload, err := sub.conn.ReadMessage()
		if websocket.IsUnexpectedCloseError(err) {
			logger.Debug(fmt.Sprintf("Closing WS connection: %s", err.Error()))
			sub.channel.Close()
			return
		}
		if err != nil {
			logger.Warn(fmt.Sprintf("Failed to read message: %s", err))
			return
		}
		msg := mainflux.Message{
			Channel:     sub.chanID,
			Subtopic:    sub.subtopic,
			ContentType: contentType,
			Publisher:   sub.pubID,
			Protocol:    protocol,
			Payload:     payload,
		}
		if err := svc.Publish(context.Background(), "", msg); err != nil {
			logger.Warn(fmt.Sprintf("Failed to publish message to NATS: %s", err))
			if err == ws.ErrFailedConnection {
				sub.conn.Close()
				sub.channel.Closed <- true
				return
			}
		}

		logger.Debug(fmt.Sprintf("Successfully published message to the channel %s", sub.chanID))
	}
}

func (sub subscription) listen() {
	for msg := range sub.channel.Messages {
		format, ok := contentTypes[msg.ContentType]
		if !ok {
			format = websocket.TextMessage
		}

		if err := sub.conn.WriteMessage(format, msg.Payload); err != nil {
			logger.Warn(fmt.Sprintf("Failed to broadcast message to thing: %s", err))
		}

		logger.Debug("Wrote message successfully")
	}
}
