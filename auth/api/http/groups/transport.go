package groups

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/internal/httputil"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go"
)

const (
	contentType = "application/json"
	maxNameSize = 254
	offsetKey   = "offset"
	limitKey    = "limit"
	levelKey    = "level"
	metadataKey = "metadata"
	treeKey     = "tree"
	groupType   = "type"
	defOffset   = 0
	defLimit    = 10
	defLevel    = 1
)

// MakeHandler returns a HTTP handler for API endpoints.
func MakeHandler(svc auth.Service, mux *bone.Mux, tracer opentracing.Tracer) *bone.Mux {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(encodeError),
	}
	mux.Post("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "create_group")(createGroupEndpoint(svc)),
		decodeGroupCreate,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "view_group")(viewGroupEndpoint(svc)),
		decodeGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Put("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "update_group")(updateGroupEndpoint(svc)),
		decodeGroupUpdate,
		encodeResponse,
		opts...,
	))

	mux.Delete("/groups/:groupID", kithttp.NewServer(
		kitot.TraceServer(tracer, "delete_group")(deleteGroupEndpoint(svc)),
		decodeGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups/:subjectGroupID/share", kithttp.NewServer(
		kitot.TraceServer(tracer, "share_group_access")(shareGroupAccessEndpoint(svc)),
		decodeShareGroupRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_groups")(listGroupsEndpoint(svc)),
		decodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID/children", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_children")(listChildrenEndpoint(svc)),
		decodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID/parents", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_parents_groups")(listParentsEndpoint(svc)),
		decodeListGroupsRequest,
		encodeResponse,
		opts...,
	))

	mux.Post("/groups/:groupID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "assign")(assignEndpoint(svc)),
		decodeAssignRequest,
		encodeResponse,
		opts...,
	))

	mux.Delete("/groups/:groupID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "unassign")(unassignEndpoint(svc)),
		decodeUnassignRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/groups/:groupID/members", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_members")(listMembersEndpoint(svc)),
		decodeListMembersRequest,
		encodeResponse,
		opts...,
	))

	mux.Get("/members/:memberID/groups", kithttp.NewServer(
		kitot.TraceServer(tracer, "list_memberships")(listMemberships(svc)),
		decodeListMembershipsRequest,
		encodeResponse,
		opts...,
	))

	return mux
}

func decodeShareGroupRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req shareGroupAccessReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.userGroupID = bone.GetValue(r, "subjectGroupID")
	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	l, err := httputil.ReadUintQuery(r, levelKey, defLevel)
	if err != nil {
		return nil, err
	}

	m, err := httputil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	t, err := httputil.ReadBoolQuery(r, treeKey, false)
	if err != nil {
		return nil, err
	}

	req := listGroupsReq{
		token:    r.Header.Get("Authorization"),
		level:    l,
		metadata: m,
		tree:     t,
		id:       bone.GetValue(r, "groupID"),
	}
	return req, nil
}

func decodeListMembersRequest(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := httputil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := httputil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := httputil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	tree, err := httputil.ReadBoolQuery(r, treeKey, false)
	if err != nil {
		return nil, err
	}

	t, err := httputil.ReadStringQuery(r, groupType, "")
	if err != nil {
		return nil, err
	}

	req := listMembersReq{
		token:     r.Header.Get("Authorization"),
		id:        bone.GetValue(r, "groupID"),
		groupType: t,
		offset:    o,
		limit:     l,
		metadata:  m,
		tree:      tree,
	}
	return req, nil
}

func decodeListMembershipsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	o, err := httputil.ReadUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := httputil.ReadUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := httputil.ReadMetadataQuery(r, metadataKey, nil)
	if err != nil {
		return nil, err
	}

	req := listMembershipsReq{
		token:    r.Header.Get("Authorization"),
		id:       bone.GetValue(r, "memberID"),
		offset:   o,
		limit:    l,
		metadata: m,
	}

	return req, nil
}

func decodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, errors.ErrUnsupportedContentType
	}

	var req updateGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	req.id = bone.GetValue(r, "groupID")
	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := groupReq{
		token: r.Header.Get("Authorization"),
		id:    bone.GetValue(r, "groupID"),
	}

	return req, nil
}

func decodeAssignRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := assignReq{
		token:   r.Header.Get("Authorization"),
		groupID: bone.GetValue(r, "groupID"),
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func decodeUnassignRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := unassignReq{
		assignReq{
			token:   r.Header.Get("Authorization"),
			groupID: bone.GetValue(r, "groupID"),
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(errors.ErrMalformedEntity, err)
	}

	return req, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", contentType)

	if ar, ok := response.(mainflux.Response); ok {
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
	switch {
	case errors.Contains(err, errors.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrAuthentication):
		w.WriteHeader(http.StatusUnauthorized)
	case errors.Contains(err, errors.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, errors.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, errors.ErrAuthorization):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, auth.ErrMemberAlreadyAssigned):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, io.EOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, io.ErrUnexpectedEOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errors.ErrUnsupportedContentType):
		w.WriteHeader(http.StatusUnsupportedMediaType)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	errorVal, ok := err.(errors.Error)
	if ok {
		if err := json.NewEncoder(w).Encode(errorRes{Err: errorVal.Msg()}); err != nil {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}
