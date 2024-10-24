// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	grpcChannelsV1 "github.com/absmach/magistrala/internal/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/internal/grpc/clients/v1"
	"github.com/absmach/magistrala/pkg/apiutil"
	mgauthn "github.com/absmach/magistrala/pkg/authn"
	"github.com/absmach/magistrala/pkg/connections"
	"github.com/absmach/magistrala/pkg/errors"
	svcerr "github.com/absmach/magistrala/pkg/errors/service"
	"github.com/absmach/magistrala/pkg/policies"
	"github.com/absmach/magistrala/readers"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	contentType    = "application/json"
	offsetKey      = "offset"
	limitKey       = "limit"
	formatKey      = "format"
	subtopicKey    = "subtopic"
	publisherKey   = "publisher"
	protocolKey    = "protocol"
	nameKey        = "name"
	valueKey       = "v"
	stringValueKey = "vs"
	dataValueKey   = "vd"
	boolValueKey   = "vb"
	comparatorKey  = "comparator"
	fromKey        = "from"
	toKey          = "to"
	aggregationKey = "aggregation"
	intervalKey    = "interval"
	defInterval    = "1s"
	defLimit       = 10
	defOffset      = 0
	defFormat      = "messages"

	tokenKind           = "token"
	clientType          = "client"
	userType            = "user"
	subscribePermission = "subscribe"
	viewPermission      = "view"
	groupType           = "group"
)

var errUserAccess = errors.New("user has no permission")

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, authn mgauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}

	mux := chi.NewRouter()
	mux.Get("/channels/{chanID}/messages", kithttp.NewServer(
		listMessagesEndpoint(svc, authn, clients, channels),
		decodeList,
		encodeResponse,
		opts...,
	).ServeHTTP)

	mux.Get("/health", magistrala.Health(svcName, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	limit, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	format, err := apiutil.ReadStringQuery(r, formatKey, defFormat)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	subtopic, err := apiutil.ReadStringQuery(r, subtopicKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	publisher, err := apiutil.ReadStringQuery(r, publisherKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	protocol, err := apiutil.ReadStringQuery(r, protocolKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	name, err := apiutil.ReadStringQuery(r, nameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	v, err := apiutil.ReadNumQuery[float64](r, valueKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	comparator, err := apiutil.ReadStringQuery(r, comparatorKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vs, err := apiutil.ReadStringQuery(r, stringValueKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vd, err := apiutil.ReadStringQuery(r, dataValueKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	vb, err := apiutil.ReadBoolQuery(r, boolValueKey, false)
	if err != nil && err != apiutil.ErrNotFoundParam {
		return nil, err
	}

	from, err := apiutil.ReadNumQuery[float64](r, fromKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	to, err := apiutil.ReadNumQuery[float64](r, toKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	aggregation, err := apiutil.ReadStringQuery(r, aggregationKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	var interval string
	if aggregation != "" {
		interval, err = apiutil.ReadStringQuery(r, intervalKey, defInterval)
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrValidation, err)
		}
	}

	req := listMessagesReq{
		chanID: chi.URLParam(r, "chanID"),
		token:  apiutil.ExtractBearerToken(r),
		key:    apiutil.ExtractClientSecret(r),
		pageMeta: readers.PageMetadata{
			Offset:      offset,
			Limit:       limit,
			Format:      format,
			Subtopic:    subtopic,
			Publisher:   publisher,
			Protocol:    protocol,
			Name:        name,
			Value:       v,
			Comparator:  comparator,
			StringValue: vs,
			DataValue:   vd,
			BoolValue:   vb,
			From:        from,
			To:          to,
			Aggregation: aggregation,
			Interval:    interval,
		},
	}
	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(magistrala.Response); ok {
		for k, v := range ar.Headers() {
			w.Header().Set(k, v)
		}

		w.WriteHeader(ar.Code())

		if ar.Empty() {
			return nil
		}
	}

	return json.NewEncoder(w).Encode(response)
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	var wrapper error
	if errors.Contains(err, apiutil.ErrValidation) {
		wrapper, err = errors.Unwrap(err)
	}

	switch {
	case errors.Contains(err, nil):
	case errors.Contains(err, apiutil.ErrInvalidQueryParams),
		errors.Contains(err, svcerr.ErrMalformedEntity),
		errors.Contains(err, apiutil.ErrMissingID),
		errors.Contains(err, apiutil.ErrLimitSize),
		errors.Contains(err, apiutil.ErrOffsetSize),
		errors.Contains(err, apiutil.ErrInvalidComparator),
		errors.Contains(err, apiutil.ErrInvalidAggregation),
		errors.Contains(err, apiutil.ErrInvalidInterval),
		errors.Contains(err, apiutil.ErrMissingFrom),
		errors.Contains(err, apiutil.ErrMissingTo),
		errors.Contains(err, apiutil.ErrMissingDomainID):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, svcerr.ErrAuthentication),
		errors.Contains(err, svcerr.ErrAuthorization),
		errors.Contains(err, apiutil.ErrBearerToken):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, readers.ErrReadMessages):
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}

	if wrapper != nil {
		err = errors.Wrap(wrapper, err)
	}
	if errorVal, ok := err.(errors.Error); ok {
		w.Header().Set("Content-Type", contentType)
		if err := json.NewEncoder(w).Encode(errorVal); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func authnAuthz(ctx context.Context, req listMessagesReq, authn mgauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient) error {
	clientID, clientType, err := authenticate(ctx, req, authn, clients)
	if err != nil {
		return nil
	}
	if err := authorize(ctx, clientID, clientType, req.chanID, channels); err != nil {
		return err
	}
	return nil
}

func authenticate(ctx context.Context, req listMessagesReq, authn mgauthn.Authentication, clients grpcClientsV1.ClientsServiceClient) (clientID string, clientType string, err error) {
	switch {
	case req.token != "":
		session, err := authn.Authenticate(ctx, req.token)
		if err != nil {
			return "", "", err
		}

		return session.DomainUserID, policies.UserType, nil
	case req.key != "":
		res, err := clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
			ClientSecret: req.key,
		})
		if err != nil {
			return "", "", err
		}
		if !res.GetAuthenticated() {
			return "", "", svcerr.ErrAuthentication
		}
		return res.GetId(), policies.ClientType, nil
	default:
		return "", "", svcerr.ErrAuthentication
	}
}

func authorize(ctx context.Context, clientID, clientType, chanID string, channels grpcChannelsV1.ChannelsServiceClient) (err error) {
	res, err := channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		ClientId:   clientID,
		ClientType: clientType,
		Type:       uint32(connections.Subscribe),
		ChannelId:  chanID,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}
	return nil
}
