// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	mggroups "github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	"github.com/absmach/magistrala/pkg/errors"
	"github.com/go-chi/chi/v5"
)

func DecodeGroupCreate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var g mggroups.Group
	if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	req := createGroupReq{
		Group: g,
	}

	return req, nil
}

func DecodeListGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	userID, err := apiutil.ReadStringQuery(r, api.UserKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	groupID, err := apiutil.ReadStringQuery(r, api.GroupKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listGroupsReq{
		PageMeta: pm,
		userID:   userID,
		groupID:  groupID,
	}
	return req, nil
}

func DecodeGroupUpdate(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := updateGroupReq{
		id: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func DecodeGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := groupReq{
		id: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func DecodeChangeGroupStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := changeGroupStatusReq{
		id: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func decodeRetrieveGroupHierarchy(_ context.Context, r *http.Request) (interface{}, error) {
	hm, err := decodeHierarchyPageMeta(r)
	if err != nil {
		return nil, err
	}

	req := retrieveGroupHierarchyReq{
		id:                chi.URLParam(r, "groupID"),
		HierarchyPageMeta: hm,
	}
	return req, nil
}

func decodeAddParentGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := addParentGroupReq{
		id: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeRemoveParentGroupRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := removeParentGroupReq{
		id: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func decodeAddChildrenGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := addChildrenGroupsReq{
		id: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeRemoveChildrenGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	req := removeChildrenGroupsReq{
		id: chi.URLParam(r, "groupID"),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(err, errors.ErrMalformedEntity))
	}
	return req, nil
}

func decodeRemoveAllChildrenGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := removeAllChildrenGroupsReq{
		id: chi.URLParam(r, "groupID"),
	}
	return req, nil
}

func decodeListChildrenGroupsRequest(_ context.Context, r *http.Request) (interface{}, error) {
	pm, err := decodePageMeta(r)
	if err != nil {
		return nil, err
	}

	startLevel, err := apiutil.ReadNumQuery[int64](r, api.StartLevelKey, api.DefStartLevel)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	endLevel, err := apiutil.ReadNumQuery[int64](r, api.EndLevelKey, api.DefEndLevel)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := listChildrenGroupsReq{
		id:         chi.URLParam(r, "groupID"),
		PageMeta:   pm,
		startLevel: startLevel,
		endLevel:   endLevel,
	}
	return req, nil
}

func decodeHierarchyPageMeta(r *http.Request) (mggroups.HierarchyPageMeta, error) {
	level, err := apiutil.ReadNumQuery[uint64](r, api.LevelKey, api.DefLevel)
	if err != nil {
		return mggroups.HierarchyPageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	tree, err := apiutil.ReadBoolQuery(r, api.TreeKey, false)
	if err != nil {
		return mggroups.HierarchyPageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	hierarchyDir, err := apiutil.ReadNumQuery[int64](r, api.DirKey, -1)
	if err != nil {
		return mggroups.HierarchyPageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	return mggroups.HierarchyPageMeta{
		Level:     level,
		Direction: hierarchyDir,
		Tree:      tree,
	}, nil
}
func decodePageMeta(r *http.Request) (mggroups.PageMeta, error) {
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefGroupStatus)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := mggroups.ToStatus(s)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	id, err := apiutil.ReadStringQuery(r, api.IDOrder, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}
	meta, err := apiutil.ReadMetadataQuery(r, api.MetadataKey, nil)
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	allActions, err := apiutil.ReadStringQuery(r, api.ActionsKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	actions := []string{}

	allActions = strings.TrimSpace(allActions)
	if allActions != "" {
		actions = strings.Split(allActions, ",")
	}
	roleID, err := apiutil.ReadStringQuery(r, api.RoleIDKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	roleName, err := apiutil.ReadStringQuery(r, api.RoleNameKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	accessType, err := apiutil.ReadStringQuery(r, api.AccessTypeKey, "")
	if err != nil {
		return mggroups.PageMeta{}, errors.Wrap(apiutil.ErrValidation, err)
	}

	ret := mggroups.PageMeta{
		Offset:     offset,
		Limit:      limit,
		Name:       name,
		ID:         id,
		Metadata:   meta,
		Status:     st,
		RoleName:   roleName,
		RoleID:     roleID,
		Actions:    actions,
		AccessType: accessType,
	}
	return ret, nil
}
