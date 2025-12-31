// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/supermq"
	grpcChannelsV1 "github.com/absmach/supermq/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/supermq/api/grpc/clients/v1"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	smqauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/connections"
	"github.com/absmach/supermq/pkg/errors"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/policies"
	"github.com/absmach/supermq/readers"
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
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc readers.MessageRepository, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(api.EncodeError),
	}

	mux := chi.NewRouter()
	mux.Get("/{domainID}/channels/{chanID}/messages", kithttp.NewServer(
		listMessagesEndpoint(svc, authn, clients, channels),
		decodeList,
		encodeResponse,
		opts...,
	).ServeHTTP)

	mux.Get("/health", supermq.Health(svcName, instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeList(_ context.Context, r *http.Request) (any, error) {
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

	order, err := apiutil.ReadStringQuery(r, api.OrderKey, "time")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	dir, err := apiutil.ReadStringQuery(r, api.DirKey, "desc")
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
		domain: chi.URLParam(r, "domainID"),
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
			Order:       order,
			Dir:         dir,
		},
	}
	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(supermq.Response); ok {
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

func authnAuthz(ctx context.Context, req listMessagesReq, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient) error {
	clientID, clientType, err := authenticate(ctx, req, authn, clients)
	if err != nil {
		return nil
	}
	if err := authorize(ctx, clientID, clientType, req.chanID, req.domain, channels); err != nil {
		return err
	}
	return nil
}

func authenticate(ctx context.Context, req listMessagesReq, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient) (clientID string, clientType string, err error) {
	switch {
	case req.token != "":
		session, err := authn.Authenticate(ctx, req.token)
		if err != nil {
			return "", "", err
		}

		return session.UserID, policies.UserType, nil
	case req.key != "":
		res, err := clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
			Token: smqauthn.AuthPack(smqauthn.DomainAuth, req.chanID, req.key),
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

func authorize(ctx context.Context, clientID, clientType, chanID, domain string, channels grpcChannelsV1.ChannelsServiceClient) (err error) {
	res, err := channels.Authorize(ctx, &grpcChannelsV1.AuthzReq{
		ClientId:   clientID,
		ClientType: clientType,
		Type:       uint32(connections.Subscribe),
		ChannelId:  chanID,
		DomainId:   domain,
	})
	if err != nil {
		return errors.Wrap(svcerr.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return svcerr.ErrAuthorization
	}
	return nil
}
