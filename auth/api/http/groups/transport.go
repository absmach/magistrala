package groups

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	kitot "github.com/go-kit/kit/tracing/opentracing"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/go-zoo/bone"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/auth"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/opentracing/opentracing-go"
)

var (
	errInvalidQueryParams     = errors.New("invalid query params")
	errUnsupportedContentType = errors.New("unsupported content type")
)

const (
	maxNameSize = 254
	offsetKey   = "offset"
	limitKey    = "limit"
	levelKey    = "level"
	metadataKey = "metadata"
	treeKey     = "tree"
	groupType   = "type"
	contentType = "application/json"

	defOffset = 0
	defLimit  = 10
	defLevel  = 1
)

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
		decodeAssignRequest,
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

func decodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, auth.ErrUnsupportedContentType
	}

	l, err := readUintQuery(r, levelKey, defLevel)
	if err != nil {
		return nil, err
	}

	m, err := readMetadataQuery(r, metadataKey)
	if err != nil {
		return nil, err
	}

	t, err := readBoolQuery(r, treeKey)
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
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, auth.ErrUnsupportedContentType
	}

	o, err := readUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := readMetadataQuery(r, metadataKey)
	if err != nil {
		return nil, err
	}

	tree, err := readBoolQuery(r, treeKey)
	if err != nil {
		return nil, err
	}

	t, err := readStringQuery(r, groupType)
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
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, auth.ErrUnsupportedContentType
	}

	o, err := readUintQuery(r, offsetKey, defOffset)
	if err != nil {
		return nil, err
	}

	l, err := readUintQuery(r, limitKey, defLimit)
	if err != nil {
		return nil, err
	}

	m, err := readMetadataQuery(r, metadataKey)
	if err != nil {
		return nil, err
	}

	tree, err := readBoolQuery(r, treeKey)
	if err != nil {
		return nil, err
	}

	req := listMembershipsReq{
		token:    r.Header.Get("Authorization"),
		id:       bone.GetValue(r, "memberID"),
		offset:   o,
		limit:    l,
		metadata: m,
		tree:     tree,
	}

	return req, nil
}

func decodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, auth.ErrUnsupportedContentType
	}

	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(auth.ErrFailedDecode, err)
	}

	req.token = r.Header.Get("Authorization")
	return req, nil
}

func decodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), contentType) {
		return nil, auth.ErrUnsupportedContentType
	}

	var req updateGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(auth.ErrFailedDecode, err)
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
		return nil, errors.Wrap(auth.ErrMalformedEntity, err)
	}

	return req, nil
}

func readUintQuery(r *http.Request, key string, def uint64) (uint64, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return 0, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return def, nil
	}

	strval := vals[0]
	val, err := strconv.ParseUint(strval, 10, 64)
	if err != nil {
		return 0, errInvalidQueryParams
	}

	return val, nil
}

func readMetadataQuery(r *http.Request, key string) (map[string]interface{}, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return nil, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return nil, nil
	}

	m := make(map[string]interface{})
	err := json.Unmarshal([]byte(vals[0]), &m)
	if err != nil {
		return nil, errors.Wrap(errInvalidQueryParams, err)
	}

	return m, nil
}

func readBoolQuery(r *http.Request, key string) (bool, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return true, errInvalidQueryParams
	}

	if len(vals) == 0 {
		return false, nil
	}

	b, err := strconv.ParseBool(vals[0])
	if err != nil {
		return false, errInvalidQueryParams
	}

	return b, nil
}

func readStringQuery(r *http.Request, key string) (string, error) {
	vals := bone.GetQuery(r, key)
	if len(vals) > 1 {
		return "", errInvalidQueryParams
	}

	if len(vals) == 0 {
		return "", nil
	}

	return vals[0], nil
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
	case errors.Contains(err, auth.ErrMalformedEntity):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, auth.ErrUnauthorizedAccess):
		w.WriteHeader(http.StatusForbidden)
	case errors.Contains(err, auth.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
	case errors.Contains(err, auth.ErrConflict):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, auth.ErrMemberAlreadyAssigned):
		w.WriteHeader(http.StatusConflict)
	case errors.Contains(err, io.EOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, io.ErrUnexpectedEOF):
		w.WriteHeader(http.StatusBadRequest)
	case errors.Contains(err, errUnsupportedContentType):
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
