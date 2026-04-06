// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	api "github.com/absmach/magistrala/api/http"
	apiutil "github.com/absmach/magistrala/api/http/util"
	"github.com/absmach/magistrala/groups"
	"github.com/absmach/magistrala/internal/nullable"
)

type createGroupReq struct {
	groups.Group
}

func (req createGroupReq) validate() error {
	if len(req.Name) > api.MaxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateGroupReq struct {
	id          string
	Name        string                 `json:"name,omitempty"`
	Description nullable.Value[string] `json:"description,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	return nil
}

type updateGroupTagsReq struct {
	id   string
	Tags []string `json:"tags,omitempty"`
}

func (req updateGroupTagsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listGroupsReq struct {
	groups.PageMeta
	userID  string
	groupID string
}

func (req listGroupsReq) validate() error {
	if req.Limit > api.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	if req.userID != "" && req.groupID != "" {
		return apiutil.ErrMultipleEntitiesFilter
	}

	switch req.Order {
	case "", api.NameOrder, api.CreatedAtOrder, api.UpdatedAtOrder:
	default:
		return apiutil.ErrInvalidOrder
	}

	if req.Dir != "" && (req.Dir != api.DescDir && req.Dir != api.AscDir) {
		return apiutil.ErrInvalidDirection
	}

	return nil
}

type groupReq struct {
	id    string
	roles bool
}

func (req groupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeGroupStatusReq struct {
	id string
}

func (req changeGroupStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type retrieveGroupHierarchyReq struct {
	groups.HierarchyPageMeta
	id string
}

func (req retrieveGroupHierarchyReq) validate() error {
	if req.Level > groups.MaxLevel {
		return apiutil.ErrLevel
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type addParentGroupReq struct {
	id       string
	ParentID string `json:"parent_id"`
}

func (req addParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if err := api.ValidateUUID(req.ParentID); err != nil {
		return err
	}
	if req.id == req.ParentID {
		return apiutil.ErrSelfParentingNotAllowed
	}
	return nil
}

type removeParentGroupReq struct {
	id string
}

func (req removeParentGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type addChildrenGroupsReq struct {
	id          string
	ChildrenIDs []string `json:"children_ids"`
}

func (req addChildrenGroupsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.ChildrenIDs) == 0 {
		return apiutil.ErrMissingChildrenGroupIDs
	}
	for _, childID := range req.ChildrenIDs {
		if err := api.ValidateUUID(childID); err != nil {
			return err
		}
		if req.id == childID {
			return apiutil.ErrSelfParentingNotAllowed
		}
	}
	return nil
}

type removeChildrenGroupsReq struct {
	id          string
	ChildrenIDs []string `json:"children_ids"`
}

func (req removeChildrenGroupsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.ChildrenIDs) == 0 {
		return apiutil.ErrMissingChildrenGroupIDs
	}
	for _, childID := range req.ChildrenIDs {
		if err := api.ValidateUUID(childID); err != nil {
			return err
		}
	}
	return nil
}

type removeAllChildrenGroupsReq struct {
	id string
}

func (req removeAllChildrenGroupsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type listChildrenGroupsReq struct {
	id         string
	startLevel int64
	endLevel   int64
	groups.PageMeta
}

func (req listChildrenGroupsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.Limit > api.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}
	return nil
}
