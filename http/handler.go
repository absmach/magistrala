// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	mgate "github.com/absmach/mgate/pkg/http"
	"github.com/absmach/mgate/pkg/session"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/absmach/supermq/pkg/policies"
)

var _ session.Handler = (*handler)(nil)

const protocol = "http"

// Log message formats.
const (
	LogInfoSubscribed   = "subscribed with client_id %s to topics %s"
	LogInfoConnected    = "connected with client_id %s"
	LogInfoDisconnected = "disconnected client_id %s and username %s"
	LogInfoPublished    = "published with client_id %s to the topic %s"
)

// Error wrappers for MQTT errors.
var (
	errClientNotInitialized     = errors.New("client is not initialized")
	errMissingTopicPub          = errors.New("failed to publish due to missing topic")
	errMissingTopicSub          = errors.New("failed to subscribe due to missing topic")
	errFailedPublish            = errors.New("failed to publish")
	errFailedPublishToMsgBroker = errors.New("failed to publish to supermq message broker")
	errInvalidAuthFormat        = errors.New("invalid basic auth format")
	errInvalidClientType        = errors.New("invalid client type")
)

// Event implements events.Event interface.
type handler struct {
	pubsub   messaging.PubSub
	clients  grpcClientsV1.ClientsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	authn    smqauthn.Authentication
	logger   *slog.Logger
	parser   messaging.TopicParser
}

// NewHandler creates new Handler entity.
func NewHandler(pubsub messaging.PubSub, logger *slog.Logger, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, parser messaging.TopicParser) session.Handler {
	return &handler{
		logger:   logger,
		pubsub:   pubsub,
		authn:    authn,
		clients:  clients,
		channels: channels,
		parser:   parser,
	}
}

// AuthConnect is called on device connection,
// prior forwarding to the http server.
func (h *handler) AuthConnect(ctx context.Context) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	var tok string
	switch {
	case string(s.Password) == "":
		return mgate.NewHTTPProxyError(http.StatusBadRequest, errors.Wrap(apiutil.ErrValidation, apiutil.ErrBearerKey))
	case strings.HasPrefix(string(s.Password), apiutil.ClientPrefix):
		tok = strings.TrimPrefix(string(s.Password), apiutil.ClientPrefix)
	default:
		tok = string(s.Password)
	}

	h.logger.Info(fmt.Sprintf(LogInfoConnected, tok))
	return nil
}

// AuthPublish is called on device publish,
// prior forwarding to the http server.
func (h *handler) AuthPublish(ctx context.Context, topic *string, payload *[]byte) error {
	if topic == nil {
		return errMissingTopicPub
	}
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	domainID, channelID, _, topicType, err := h.parser.ParsePublishTopic(ctx, *topic, true)
	if err != nil {
		return mgate.NewHTTPProxyError(http.StatusBadRequest, errors.Wrap(errFailedPublish, err))
	}

	clientID, err := h.authAccess(ctx, s.Username, string(s.Password), domainID, channelID, connections.Publish, topicType)
	if err != nil {
		return err
	}

	if s.Username == "" {
		s.Username = clientID
	}

	return nil
}

// AuthSubscribe is called on device publish,
// prior forwarding to the MQTT broker.
func (h *handler) AuthSubscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}
	if topics == nil || *topics == nil {
		return errMissingTopicSub
	}

	for _, topic := range *topics {
		domainID, channelID, _, topicType, err := h.parser.ParseSubscribeTopic(ctx, topic, true)
		if err != nil {
			return err
		}
		if _, err := h.authAccess(ctx, s.Username, string(s.Password), domainID, channelID, connections.Subscribe, topicType); err != nil {
			return err
		}
	}
	return nil
}

// Connect - after client successfully connected.
func (h *handler) Connect(ctx context.Context) error {
	return nil
}

// Publish - after client successfully published.
func (h *handler) Publish(ctx context.Context, topic *string, payload *[]byte) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}

	if len(*payload) == 0 {
		h.logger.Warn("Empty payload, not publishing to broker", slog.String("client_id", s.Username))
		return nil
	}

	domainID, channelID, subtopic, topicType, err := h.parser.ParsePublishTopic(ctx, *topic, true)
	if err != nil {
		return errors.Wrap(errFailedPublish, err)
	}

	msg := messaging.Message{
		Protocol:  protocol,
		Domain:    domainID,
		Channel:   channelID,
		Subtopic:  subtopic,
		Payload:   *payload,
		Publisher: s.Username,
		Created:   time.Now().UnixNano(),
	}

	// Health check topic messages do not get published to message broker.
	if topicType == messaging.MessageType {
		if err := h.pubsub.Publish(ctx, messaging.EncodeMessageTopic(&msg), &msg); err != nil {
			return mgate.NewHTTPProxyError(http.StatusInternalServerError, errors.Wrap(errFailedPublishToMsgBroker, err))
		}
	}

	h.logger.Info(fmt.Sprintf(LogInfoPublished, s.ID, *topic))

	return nil
}

// Subscribe - after client successfully subscribed.
func (h *handler) Subscribe(ctx context.Context, topics *[]string) error {
	s, ok := session.FromContext(ctx)
	if !ok {
		return errClientNotInitialized
	}
	h.logger.Info(fmt.Sprintf(LogInfoSubscribed, s.ID, strings.Join(*topics, ",")))
	return nil
}

// Unsubscribe - after client unsubscribed.
func (h *handler) Unsubscribe(ctx context.Context, topics *[]string) error {
	return nil
}

// Disconnect - connection with broker or client lost.
func (h *handler) Disconnect(ctx context.Context) error {
	return nil
}

func (h *handler) authAccess(ctx context.Context, username, password, domainID, chanID string, msgType connections.ConnType, topicType messaging.TopicType) (string, error) {
	var token, clientType string
	var err error
	switch {
	case strings.HasPrefix(password, apiutil.BearerPrefix):
		token = strings.TrimPrefix(password, apiutil.BearerPrefix)
		clientType = policies.UserType
	case username != "" && password != "":
		token = smqauthn.AuthPack(smqauthn.BasicAuth, username, password)
		clientType = policies.ClientType
	case strings.HasPrefix(password, apiutil.BasicAuthPrefix):
		username, password, err := decodeAuth(strings.TrimPrefix(password, apiutil.BasicAuthPrefix))
		if err != nil {
			return "", errors.Wrap(svcerr.ErrAuthentication, err)
		}
		token = smqauthn.AuthPack(smqauthn.BasicAuth, username, password)
		clientType = policies.ClientType
	default:
		token = smqauthn.AuthPack(smqauthn.DomainAuth, domainID, strings.TrimPrefix(password, apiutil.ClientPrefix))
		clientType = policies.ClientType
	}

	id, subject, err := h.authenticate(ctx, clientType, token, domainID)
	if err != nil {
		return "", mgate.NewHTTPProxyError(http.StatusUnauthorized, errors.Wrap(svcerr.ErrAuthentication, err))
	}

	// Health check topics do not require channel authorization.
	if topicType == messaging.HealthType {
		return id, nil
	}

	ar := &grpcChannelsV1.AuthzReq{
		Type:       uint32(msgType),
		ClientId:   subject,
		ClientType: clientType,
		ChannelId:  chanID,
		DomainId:   domainID,
	}
	res, err := h.channels.Authorize(ctx, ar)
	if err != nil {
		return "", mgate.NewHTTPProxyError(http.StatusUnauthorized, errors.Wrap(svcerr.ErrAuthentication, err))
	}
	if !res.GetAuthorized() {
		return "", mgate.NewHTTPProxyError(http.StatusUnauthorized, svcerr.ErrAuthentication)
	}

	return id, nil
}

func (h *handler) authenticate(ctx context.Context, authType, token, domainID string) (string, string, error) {
	switch authType {
	case policies.UserType:
		authnSession, err := h.authn.Authenticate(ctx, token)
		if err != nil {
			return "", "", err
		}
		if authnSession.Role == smqauthn.SuperAdminRole {
			return authnSession.UserID, authnSession.UserID, nil
		}
		return authnSession.UserID, policies.EncodeDomainUserID(domainID, authnSession.UserID), nil
	case policies.ClientType:
		authnRes, err := h.clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{Token: token})
		if err != nil {
			return "", "", err
		}
		if !authnRes.Authenticated {
			return "", "", svcerr.ErrAuthentication
		}

		return authnRes.GetId(), authnRes.GetId(), nil
	default:
		return "", "", errInvalidClientType
	}
}

// decodeAuth decodes the base64 encoded string in the format "clientID:secret".
func decodeAuth(s string) (string, string, error) {
	db, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(db), ":", 2)
	if len(parts) != 2 {
		return "", "", errInvalidAuthFormat
	}
	clientID := parts[0]
	secret := parts[1]

	return clientID, secret, nil
}
