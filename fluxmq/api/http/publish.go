// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

// Package http exposes user-authenticated message publishing for the MG UI.
package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/absmach/magistrala/internal/atom"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/go-chi/chi/v5"
)

const (
	contentType = "application/json"
	httpProto   = "http"
)

type publishRequest struct {
	ClientID string          `json:"client_id"`
	Subtopic string          `json:"subtopic"`
	Payload  json.RawMessage `json:"payload"`
}

type publishResponse struct {
	Status string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type publishHandler struct {
	authn     smqauthn.Authentication
	atom      *atom.Client
	publisher messaging.Publisher
}

// MakePublishHandler returns an HTTP handler that authenticates the user with
// Atom, authorizes publish access in Atom, and writes directly to the message
// broker with the selected client as the publisher identity.
func MakePublishHandler(
	authn smqauthn.Authentication,
	atomClient *atom.Client,
	publisher messaging.Publisher,
) http.Handler {
	h := publishHandler{
		authn:     authn,
		atom:      atomClient,
		publisher: publisher,
	}
	r := chi.NewRouter()
	r.Post("/{domainID}/channels/{channelID}/messages", h.publish)
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(publishResponse{Status: "ok"}); err != nil {
			return
		}
	})
	return r
}

func (h publishHandler) publish(w http.ResponseWriter, r *http.Request) {
	domainID := chi.URLParam(r, "domainID")
	channelID := chi.URLParam(r, "channelID")
	if domainID == "" || channelID == "" {
		writeError(w, http.StatusBadRequest, "domainID and channelID are required")
		return
	}

	token := bearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "bearer token is required")
		return
	}
	session, err := h.authn.Authenticate(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid bearer token")
		return
	}

	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid publish request")
		return
	}
	payload, err := payloadBytes(req.Payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	publisherID := session.UserID
	if req.ClientID != "" {
		if err := h.ensureClientPublisher(r.Context(), domainID, channelID, session.UserID, req.ClientID); err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		publisherID = req.ClientID
	}
	if err := h.ensureUserPublish(r.Context(), domainID, channelID, session.UserID, req.ClientID); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	subtopic := cleanSubtopic(req.Subtopic)
	topic := messaging.EncodeTopicSuffix(domainID, channelID, subtopic)

	msg := &messaging.Message{
		Domain:    domainID,
		Channel:   channelID,
		Subtopic:  subtopic,
		Publisher: publisherID,
		ClientId:  session.UserID,
		Protocol:  httpProto,
		Payload:   payload,
		Created:   time.Now().UnixNano(),
	}
	if err := h.publisher.Publish(r.Context(), topic, msg); err != nil {
		writeError(w, http.StatusBadGateway, "failed to publish message")
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(publishResponse{Status: "accepted"}); err != nil {
		return
	}
}

func (h publishHandler) ensureUserPublish(
	ctx context.Context,
	domainID string,
	channelID string,
	userID string,
	clientID string,
) error {
	res, err := h.atom.CheckAuthz(ctx, atom.AuthzRequest{
		SubjectID:  userID,
		Action:     "publish",
		ResourceID: channelID,
		ObjectKind: "resource",
		ObjectID:   channelID,
		Context: map[string]any{
			"domain_id":           domainID,
			"publisher_client_id": clientID,
		},
	})
	if err != nil {
		return err
	}
	if !res.Allowed {
		return fmt.Errorf("user is not allowed to publish to channel")
	}
	return nil
}

func (h publishHandler) ensureClientPublisher(
	ctx context.Context,
	domainID string,
	channelID string,
	userID string,
	clientID string,
) error {
	client, err := h.atom.GetEntity(ctx, clientID)
	if err != nil {
		return fmt.Errorf("publisher client not found")
	}
	if client.Kind != "device" && attrString(client.Attributes, "magistrala_kind") != atom.KindClient {
		return fmt.Errorf("publisher identity is not a client")
	}
	if client.TenantID == "" || client.TenantID != domainID {
		return fmt.Errorf("publisher client belongs to a different domain")
	}
	userAccess, err := h.atom.CheckAuthz(ctx, atom.AuthzRequest{
		SubjectID:  userID,
		Action:     "read",
		ResourceID: clientID,
		ObjectKind: "entity",
		ObjectID:   clientID,
		Context: map[string]any{
			"domain_id": domainID,
		},
	})
	if err != nil {
		return err
	}
	if !userAccess.Allowed {
		return fmt.Errorf("user is not allowed to use publisher client")
	}
	res, err := h.atom.CheckAuthz(ctx, atom.AuthzRequest{
		SubjectID:  clientID,
		Action:     "publish",
		ResourceID: channelID,
		ObjectKind: "resource",
		ObjectID:   channelID,
		Context: map[string]any{
			"domain_id": domainID,
		},
	})
	if err != nil {
		return err
	}
	if !res.Allowed {
		return fmt.Errorf("publisher client is not connected for publish")
	}
	return nil
}

func bearerToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	return strings.TrimPrefix(token, "Bearer ")
}

func payloadBytes(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("payload is required")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("payload must be a string or JSON value")
		}
		return []byte(value), nil
	}
	return raw, nil
}

func cleanSubtopic(subtopic string) string {
	return strings.Trim(strings.ReplaceAll(subtopic, ".", "/"), "/")
}

func attrString(attrs atom.Attributes, key string) string {
	value, ok := attrs[key]
	if !ok || value == nil {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return fmt.Sprint(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		return
	}
}
