// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	stderrors "errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	authv1 "github.com/absmach/fluxmq/pkg/proto/auth/v1"
	"github.com/absmach/fluxmq/pkg/proto/auth/v1/authv1connect"
	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	apiutil "github.com/absmach/magistrala/api/http/util"
	smqauth "github.com/absmach/magistrala/auth"
	"github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/policies"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ authv1connect.AuthServiceHandler = (*connectServer)(nil)

const (
	transientRetryAttempts = 2
	transientRetryBackoff  = 75 * time.Millisecond
)

type connectServer struct {
	authv1connect.UnimplementedAuthServiceHandler
	clients  grpcClientsV1.ClientsServiceClient
	channels grpcChannelsV1.ChannelsServiceClient
	parser   messaging.TopicParser
}

// NewServer creates a FluxMQ AuthService Connect handler that bridges to
// Magistrala's Clients (authn) and Channels (authz) services.
func NewServer(
	clients grpcClientsV1.ClientsServiceClient,
	channels grpcChannelsV1.ChannelsServiceClient,
	parser messaging.TopicParser,
) authv1connect.AuthServiceHandler {
	return &connectServer{
		clients:  clients,
		channels: channels,
		parser:   parser,
	}
}

func (s *connectServer) Authenticate(ctx context.Context, req *connect.Request[authv1.AuthnReq]) (*connect.Response[authv1.AuthnRes], error) {
	username := req.Msg.GetUsername()
	password := req.Msg.GetPassword()

	token := authn.AuthPack(authn.BasicAuth, username, password)
	res, err := withTransientRetry(ctx, func(callCtx context.Context) (*grpcClientsV1.AuthnRes, error) {
		return s.clients.Authenticate(callCtx, &grpcClientsV1.AuthnReq{Token: token})
	})
	if err != nil {
		if !shouldTryDomainAuth(req.Msg, username, password) {
			return nil, encodeError(err)
		}

		token = authn.AuthPack(authn.DomainAuth, username, password)
		res, err = withTransientRetry(ctx, func(callCtx context.Context) (*grpcClientsV1.AuthnRes, error) {
			return s.clients.Authenticate(callCtx, &grpcClientsV1.AuthnReq{Token: token})
		})
		if err != nil {
			return nil, encodeError(err)
		}
	}

	return connect.NewResponse(&authv1.AuthnRes{
		Authenticated: res.GetAuthenticated(),
		Id:            res.GetId(),
	}), nil
}

func (s *connectServer) Authorize(ctx context.Context, req *connect.Request[authv1.AuthzReq]) (*connect.Response[authv1.AuthzRes], error) {
	connType := connections.ConnType(req.Msg.GetAction())
	if err := connections.CheckConnType(connType); err != nil {
		return nil, encodeError(err)
	}

	var domainID, channelID string
	var topicType messaging.TopicType
	var err error

	type parseResult struct {
		domainID  string
		channelID string
		topicType messaging.TopicType
	}
	var parsed parseResult

	switch connType {
	case connections.Publish:
		parsed, err = withTransientRetry(ctx, func(callCtx context.Context) (parseResult, error) {
			domainID, channelID, _, topicType, err := s.parser.ParsePublishTopic(callCtx, req.Msg.GetTopic(), true)
			if err != nil {
				return parseResult{}, err
			}
			return parseResult{domainID: domainID, channelID: channelID, topicType: topicType}, nil
		})
	case connections.Subscribe:
		parsed, err = withTransientRetry(ctx, func(callCtx context.Context) (parseResult, error) {
			domainID, channelID, _, topicType, err := s.parser.ParseSubscribeTopic(callCtx, req.Msg.GetTopic(), true)
			if err != nil {
				return parseResult{}, err
			}
			return parseResult{domainID: domainID, channelID: channelID, topicType: topicType}, nil
		})
	}
	if err != nil {
		if shouldDenyAuthorize(err) {
			return connect.NewResponse(&authv1.AuthzRes{Authorized: false}), nil
		}
		return nil, encodeError(err)
	}

	domainID = parsed.domainID
	channelID = parsed.channelID
	topicType = parsed.topicType

	if topicType == messaging.HealthType {
		return connect.NewResponse(&authv1.AuthzRes{Authorized: true}), nil
	}

	ar := &grpcChannelsV1.AuthzReq{
		Type:       uint32(connType),
		ClientId:   req.Msg.GetExternalId(),
		ClientType: policies.ClientType,
		ChannelId:  channelID,
		DomainId:   domainID,
	}
	res, err := withTransientRetry(ctx, func(callCtx context.Context) (*grpcChannelsV1.AuthzRes, error) {
		return s.channels.Authorize(callCtx, ar)
	})
	if err != nil {
		if shouldDenyAuthorize(err) {
			return connect.NewResponse(&authv1.AuthzRes{Authorized: false}), nil
		}
		return nil, encodeError(err)
	}

	return connect.NewResponse(&authv1.AuthzRes{
		Authorized: res.GetAuthorized(),
	}), nil
}

func shouldTryDomainAuth(msg *authv1.AuthnReq, username, password string) bool {
	if username == "" || password == "" {
		return false
	}

	return strings.HasPrefix(msg.GetClientId(), "http:")
}

func shouldDenyAuthorize(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, svcerr.ErrNotFound),
		errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, messaging.ErrMalformedTopic),
		err == apiutil.ErrMissingID:
		return true
	}

	// Backward compatibility for gRPC client layers that may return
	// Internal with a payload containing "entity not found".
	return strings.Contains(err.Error(), svcerr.ErrNotFound.Error())
}

func withTransientRetry[T any](ctx context.Context, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	var lastErr error

	for attempt := 0; attempt < transientRetryAttempts; attempt++ {
		res, err := fn(ctx)
		if err == nil {
			return res, nil
		}
		lastErr = err

		if attempt == transientRetryAttempts-1 || !isTransientError(err) {
			return zero, err
		}

		if !sleepWithContext(ctx, transientRetryBackoff) {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return zero, ctxErr
			}
			return zero, err
		}
	}

	return zero, lastErr
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Permanent errors must not be retried.
	switch {
	case shouldDenyAuthorize(err),
		errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, smqauth.ErrKeyExpired),
		errors.Contains(err, errors.ErrMalformedEntity),
		errors.Contains(err, messaging.ErrMalformedTopic),
		err == apiutil.ErrMissingID:
		return false
	}

	var connectErr *connect.Error
	if stderrors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnavailable, connect.CodeDeadlineExceeded, connect.CodeCanceled, connect.CodeAborted, connect.CodeResourceExhausted:
			return true
		default:
			return false
		}
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.Aborted, codes.ResourceExhausted:
			return true
		default:
			return false
		}
	}

	msg := strings.ToLower(err.Error())
	retryableFragments := []string{
		"unavailable",
		"deadline exceeded",
		"timed out",
		"timeout",
		"eof",
		"connection reset",
		"broken pipe",
		"connection refused",
		"transport is closing",
		"http2: client connection lost",
		"server is closing",
	}
	for _, frag := range retryableFragments {
		if strings.Contains(msg, frag) {
			return true
		}
	}

	return false
}

func encodeError(err error) error {
	switch {
	case errors.Contains(err, nil):
		return nil
	case errors.Contains(err, errors.ErrMalformedEntity),
		err == apiutil.ErrMissingID:
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, smqauth.ErrKeyExpired):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Contains(err, svcerr.ErrAuthorization):
		return connect.NewError(connect.CodePermissionDenied, err)
	case errors.Contains(err, messaging.ErrMalformedTopic):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
