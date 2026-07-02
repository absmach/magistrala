// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/pkg/messaging"
)

const (
	hookResultOK   = "ok"
	hookResultDeny = "deny"

	hookAuthOnPublish     = "auth_on_publish"
	hookAuthOnSubscribe   = "auth_on_subscribe"
	hookAuthOnUnsubscribe = "auth_on_unsubscribe"
)

type hookRequest struct {
	Hook       string            `json:"hook"`
	ClientID   string            `json:"client_id"`
	ExternalID string            `json:"external_id"`
	Protocol   string            `json:"protocol"`
	Topic      string            `json:"topic"`
	Payload    []byte            `json:"payload,omitempty"`
	QoS        uint32            `json:"qos"`
	Retain     bool              `json:"retain"`
	Properties map[string]string `json:"properties,omitempty"`
	Username   string            `json:"username,omitempty"`
	Password   string            `json:"password,omitempty"`
}

type hookResponse struct {
	Result     string            `json:"result"`
	Topic      string            `json:"topic,omitempty"`
	Payload    []byte            `json:"payload,omitempty"`
	PayloadSet bool              `json:"payload_set,omitempty"`
	QoS        uint32            `json:"qos,omitempty"`
	QoSSet     bool              `json:"qos_set,omitempty"`
	Retain     bool              `json:"retain,omitempty"`
	RetainSet  bool              `json:"retain_set,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
	ExternalID string            `json:"external_id,omitempty"`
	ReasonCode uint32            `json:"reason_code,omitempty"`
	Reason     string            `json:"reason,omitempty"`
}

// MakeHooksHandler returns an HTTP handler for FluxMQ blocking hooks.
func MakeHooksHandler(parser messaging.TopicParser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req hookRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid hook request")
			return
		}

		res := handleHook(r.Context(), parser, req)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(res); err != nil {
			return
		}
	})
}

func handleHook(ctx context.Context, parser messaging.TopicParser, req hookRequest) hookResponse {
	topic, err := resolveHookTopic(ctx, parser, req)
	if err != nil {
		return hookResponse{Result: hookResultDeny, Reason: err.Error()}
	}
	return hookResponse{Result: hookResultOK, Topic: topic}
}

func resolveHookTopic(ctx context.Context, parser messaging.TopicParser, req hookRequest) (string, error) {
	if !isMessageTopic(req.Topic) {
		return "", nil
	}
	if parser == nil {
		return "", fmt.Errorf("topic parser is not configured")
	}

	var domainID, channelID, subtopic string
	var topicType messaging.TopicType
	var err error

	switch strings.ToLower(strings.TrimSpace(req.Hook)) {
	case hookAuthOnPublish:
		domainID, channelID, subtopic, topicType, err = parser.ParsePublishTopic(ctx, req.Topic, true)
	case hookAuthOnSubscribe, hookAuthOnUnsubscribe:
		domainID, channelID, subtopic, topicType, err = parser.ParseSubscribeTopic(ctx, req.Topic, true)
	default:
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if topicType != messaging.MessageType {
		return "", nil
	}

	return messaging.EncodeTopic(domainID, channelID, subtopic), nil
}

func isMessageTopic(topic string) bool {
	topic = strings.TrimSpace(topic)
	topic = strings.TrimPrefix(topic, "/")
	return strings.HasPrefix(topic, string(messaging.MsgTopicPrefix)+"/")
}
