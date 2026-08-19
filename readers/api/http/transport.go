// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/absmach/magistrala"
	grpcChannelsV1 "github.com/absmach/magistrala/api/grpc/channels/v1"
	grpcClientsV1 "github.com/absmach/magistrala/api/grpc/clients/v1"
	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	atomevents "github.com/absmach/magistrala/pkg/atom/events"
	smqauthn "github.com/absmach/magistrala/pkg/authn"
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
	publishersKey  = "publishers"
	deviceIDKey    = "device_id"
	deviceIDsKey   = "device_ids"
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

// MakeHandler returns a HTTP handler for API endpoints. invalidation may be
// nil -- e.g. when the Atom events broker is unreachable or unconfigured, see
// cmd/postgres-reader and cmd/timescale-reader -- in which case readAuthz
// falls back to pure TTL-based expiry exactly as it did before MG-14; event-
// driven invalidation is an optimization layered on top of that TTL, never a
// substitute for it.
func MakeHandler(svc readers.MessageRepository, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, policyEvaluator policies.Evaluator, policyLister policies.Service, devices externalIDResolver, invalidation *atomevents.Registry, svcName, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(api.EncodeError),
	}

	readAuthz := newReadAuthorizer(policyEvaluator, policyLister, devices)
	if invalidation != nil {
		invalidation.Register(atomevents.FamilyTranslation, readAuthz)
		invalidation.Register(atomevents.FamilyAuthorizedSet, readAuthz)
	}

	mux := chi.NewRouter()
	mux.Get("/{domainID}/channels/{chanID}/messages", kithttp.NewServer(
		listMessagesEndpoint(svc, authn, clients, channels, readAuthz),
		decodeList,
		encodeResponse,
		opts...,
	).ServeHTTP)

	// MG-15 observed-device aggregation. Both directions are nested under
	// the channel, not under new top-level /gateways or /devices resources:
	// every read here is channel-scoped (readers hold no notion of a
	// gateway or device outside a channel's messages, and the only
	// authorization check available is the channel-level subscribe check
	// authorize() already performs), and a gateway's publisher id is only
	// meaningful within one channel's rows in this store. Nesting under
	// /channels/{chanID} keeps that scoping explicit in the URL, mirroring
	// the existing messages route.
	//
	// The anchoring identity (the gateway's publisher id, or the device's
	// serial) is a query parameter, not a URL path segment, on both routes.
	// A device serial is format-unconstrained (MG-09: Atom's external_id
	// accepts `/`, spaces and unicode — see readRepeatedQuery below, which
	// exists for the same reason), so putting one in a path segment risks
	// chi splitting it as extra path segments or requiring escaping a caller
	// has to know to apply. Keeping both directions consistent rather than
	// only routing device_id this way avoids that asymmetry looking
	// accidental.
	mux.Get("/{domainID}/channels/{chanID}/devices", kithttp.NewServer(
		listGatewayDevicesEndpoint(svc, authn, clients, channels, readAuthz),
		decodeGatewayDevices,
		encodeResponse,
		opts...,
	).ServeHTTP)
	mux.Get("/{domainID}/channels/{chanID}/publishers", kithttp.NewServer(
		listDeviceGatewaysEndpoint(svc, authn, clients, channels, readAuthz),
		decodeDeviceGateways,
		encodeResponse,
		opts...,
	).ServeHTTP)

	mux.Get("/health", magistrala.Health(svcName, instanceID))
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

	publishers := readRepeatedQuery(r, publishersKey)
	deviceIDs := readRepeatedQuery(r, deviceIDsKey)

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
			Publishers:  publishers,
			DeviceIDs:   deviceIDs,
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

func decodeGatewayDevices(_ context.Context, r *http.Request) (any, error) {
	return decodeDeviceView(r, publisherKey, true)
}

func decodeDeviceGateways(_ context.Context, r *http.Request) (any, error) {
	return decodeDeviceView(r, deviceIDKey, false)
}

// decodeDeviceView decodes the shared request shape of both MG-15
// observed-device endpoints. idKey names the query parameter holding the
// fixed gateway publisher id or device serial for the given direction;
// filterIsPublisher records which one it is, so that validate() can check a
// publisher id for UUID well-formedness while leaving serials unconstrained.
func decodeDeviceView(r *http.Request, idKey string, filterIsPublisher bool) (any, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, offsetKey, defOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	limit, err := apiutil.ReadNumQuery[uint64](r, limitKey, defLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	filterVal, err := apiutil.ReadStringQuery(r, idKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	from, err := apiutil.ReadNumQuery[float64](r, fromKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	to, err := apiutil.ReadNumQuery[float64](r, toKey, 0)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	from, to = readers.DefaultTimeWindow(from, to)

	req := deviceViewReq{
		chanID:            chi.URLParam(r, "chanID"),
		token:             apiutil.ExtractBearerToken(r),
		domain:            chi.URLParam(r, "domainID"),
		key:               apiutil.ExtractClientSecret(r),
		filterVal:         filterVal,
		filterIsPublisher: filterIsPublisher,
		pageMeta: readers.PageMetadata{
			Offset: offset,
			Limit:  limit,
			From:   from,
			To:     to,
		},
	}
	return req, nil
}

// readRepeatedQuery collects a list-valued filter from repeated query parameters
// — ?device_ids=A&device_ids=B — and drops empty entries.
//
// Values are taken verbatim, never split on a separator. Device serials carry no
// format constraint at all (MG-09; Atom's external_id accepts `/`, spaces and
// unicode), so any separator we chose could occur inside a legitimate serial and
// splitting on it would silently query for something the caller never asked for.
func readRepeatedQuery(r *http.Request, key string) []string {
	raw := r.URL.Query()[key]
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response any) error {
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

// authnAuthz authenticates the caller, applies the channel-level subscribe check, then bounds the
// query to the devices the caller is authorized to read and narrows their own filters to the same
// set. noAccess means the caller may read nothing on this channel, so the response must be empty
// rather than unfiltered.
func authnAuthz(ctx context.Context, req listMessagesReq, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient, channels grpcChannelsV1.ChannelsServiceClient, readAuthz *readAuthorizer) (pm readers.PageMetadata, noAccess bool, err error) {
	clientID, clientType, superAdmin, err := authenticate(ctx, req.token, req.key, req.domain, authn, clients)
	if err != nil {
		return readers.PageMetadata{}, false, err
	}
	if err := authorize(ctx, clientID, clientType, req.chanID, req.domain, channels); err != nil {
		return readers.PageMetadata{}, false, err
	}

	if superAdmin {
		return req.pageMeta, false, nil
	}

	// Per-device grants only exist for domain users, not for clients authenticating with a secret key.
	if clientType != policies.UserType {
		return req.pageMeta, false, nil
	}

	scope, err := readAuthz.resolve(ctx, req.domain, clientID, requestedPublishers(req.pageMeta), requestedDeviceIDs(req.pageMeta))
	if err != nil {
		return readers.PageMetadata{}, false, err
	}
	if scope.noAccess {
		return req.pageMeta, true, nil
	}

	pm = req.pageMeta
	pm.Publisher = ""
	pm.Publishers = scope.publishers
	pm.DeviceIDs = scope.deviceIDs
	pm.DeviceScope = scope.scope
	return pm, false, nil
}

func authenticate(ctx context.Context, token, key, domain string, authn smqauthn.Authentication, clients grpcClientsV1.ClientsServiceClient) (clientID string, clientType string, superAdmin bool, err error) {
	switch {
	case token != "":
		session, err := authn.Authenticate(ctx, token)
		if err != nil {
			return "", "", false, err
		}
		if session.Role == smqauthn.SuperAdminRole {
			return session.UserID, policies.UserType, true, nil
		}

		return policies.EncodeDomainUserID(domain, session.UserID), policies.UserType, false, nil
	case key != "":
		res, err := clients.Authenticate(ctx, &grpcClientsV1.AuthnReq{
			Token: smqauthn.AuthPack(smqauthn.DomainAuth, domain, key),
		})
		if err != nil {
			return "", "", false, err
		}
		if !res.GetAuthenticated() {
			return "", "", false, svcerr.ErrAuthentication
		}
		return res.GetId(), policies.ClientType, false, nil
	default:
		return "", "", false, svcerr.ErrAuthentication
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
