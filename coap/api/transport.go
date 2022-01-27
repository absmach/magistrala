// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"

	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/coap"
	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/plgd-dev/go-coap/v2/message"
	"github.com/plgd-dev/go-coap/v2/message/codes"
	"github.com/plgd-dev/go-coap/v2/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	protocol  = "coap"
	authQuery = "auth"
)

var channelPartRegExp = regexp.MustCompile(`^channels/([\w\-]+)/messages(/[^?]*)?(\?.*)?$`)

var errMalformedSubtopic = errors.New("malformed subtopic")

var (
	logger  log.Logger
	service coap.Service
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHTTPHandler() http.Handler {
	b := bone.New()
	b.GetFunc("/health", mainflux.Health(protocol))
	b.Handle("/metrics", promhttp.Handler())

	return b
}

// MakeCoAPHandler creates handler for CoAP messages.
func MakeCoAPHandler(svc coap.Service, l log.Logger) mux.HandlerFunc {
	logger = l
	service = svc

	return handler
}

func sendResp(w mux.ResponseWriter, resp *message.Message) {
	if err := w.Client().WriteMessage(resp); err != nil {
		logger.Warn(fmt.Sprintf("Can't set response: %s", err))
	}
}

func handler(w mux.ResponseWriter, m *mux.Message) {
	resp := message.Message{
		Code:    codes.Content,
		Token:   m.Token,
		Context: m.Context,
		Options: make(message.Options, 0, 16),
	}
	defer sendResp(w, &resp)
	if m.Options == nil {
		logger.Warn("Nil options")
		resp.Code = codes.BadOption
		return
	}
	msg, err := decodeMessage(m)
	if err != nil {
		logger.Warn(fmt.Sprintf("Error decoding message: %s", err))
		resp.Code = codes.BadRequest
		return
	}
	key, err := parseKey(m)
	if err != nil {
		logger.Warn(fmt.Sprintf("Error parsing auth: %s", err))
		resp.Code = codes.Unauthorized
		return
	}
	switch m.Code {
	case codes.GET:
		var obs uint32
		obs, err = m.Options.Observe()
		if err != nil {
			resp.Code = codes.BadOption
			logger.Warn(fmt.Sprintf("Error reading observe option: %s", err))
			return
		}
		if obs == 0 {
			c := coap.NewClient(w.Client(), m.Token, logger)
			err = service.Subscribe(context.Background(), key, msg.Channel, msg.Subtopic, c)
			break
		}
		service.Unsubscribe(context.Background(), key, msg.Channel, msg.Subtopic, m.Token.String())
	case codes.POST:
		err = service.Publish(context.Background(), key, msg)
	default:
		resp.Code = codes.NotFound
		return
	}
	if err != nil {
		switch {
		case errors.Contains(err, errors.ErrAuthorization):
			resp.Code = codes.Unauthorized
			return
		case errors.Contains(err, coap.ErrUnsubscribe):
			resp.Code = codes.InternalServerError
		}
	}
}

func decodeMessage(msg *mux.Message) (messaging.Message, error) {
	path, err := msg.Options.Path()
	if err != nil {
		return messaging.Message{}, err
	}
	channelParts := channelPartRegExp.FindStringSubmatch(path)
	if len(channelParts) < 2 {
		return messaging.Message{}, errMalformedSubtopic
	}

	st, err := parseSubtopic(channelParts[2])
	if err != nil {
		return messaging.Message{}, err
	}
	ret := messaging.Message{
		Protocol: protocol,
		Channel:  parseID(path),
		Subtopic: st,
		Payload:  []byte{},
		Created:  time.Now().UnixNano(),
	}

	if msg.Body != nil {
		buff, err := ioutil.ReadAll(msg.Body)
		if err != nil {
			return ret, err
		}
		ret.Payload = buff
	}
	return ret, nil
}

func parseID(path string) string {
	vars := strings.Split(path, "/")
	if len(vars) > 1 {
		return vars[1]
	}
	return ""
}

func parseKey(msg *mux.Message) (string, error) {
	authKey, err := msg.Options.GetString(message.URIQuery)
	if err != nil {
		return "", err
	}
	vars := strings.Split(authKey, "=")
	if len(vars) != 2 || vars[0] != authQuery {
		return "", errors.ErrAuthorization
	}
	return vars[1], nil
}

func parseSubtopic(subtopic string) (string, error) {
	if subtopic == "" {
		return subtopic, nil
	}

	subtopic, err := url.QueryUnescape(subtopic)
	if err != nil {
		return "", errMalformedSubtopic
	}
	subtopic = strings.ReplaceAll(subtopic, "/", ".")

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
