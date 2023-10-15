// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/mainflux/mainflux/internal/api"
	"github.com/mainflux/mainflux/internal/apiutil"
	mfgroups "github.com/mainflux/mainflux/pkg/groups"
)

const (
	thingsKind = "things"
)

type createGroupReq struct {
	mfgroups.Group
	token string
}

func (req createGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Name) > api.MaxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateGroupReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if len(req.Name) > api.MaxNameSize {
		return apiutil.ErrNameSize
	}
	return nil
}

type listGroupsReq struct {
	mfgroups.Page
	token      string
	memberKind string
	memberID   string
	// - `true`  - result is JSON tree representing groups hierarchy,
	// - `false` - result is JSON array of groups.
	tree bool
}

func (req listGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.memberKind == "" {
		return apiutil.ErrMissingMemberKind
	}
	if req.memberKind == thingsKind && req.memberID == "" {
		return apiutil.ErrMissingID
	}
	if req.Level < mfgroups.MinLevel || req.Level > mfgroups.MaxLevel {
		return apiutil.ErrInvalidLevel
	}

	return nil
}

type listMembershipReq struct {
	mfgroups.Page
	token      string
	groupID    string
	memberKind string
}

func (req listMembershipReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if req.memberKind == "" {
		return apiutil.ErrMissingMemberKind
	}
	return nil
}

type groupReq struct {
	token string
	id    string
}

func (req groupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type changeGroupStatusReq struct {
	token string
	id    string
}

func (req changeGroupStatusReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type assignReq struct {
	token      string
	groupID    string
	Relation   string   `json:"relation,omitempty"`
	MemberKind string   `json:"member_kind,omitempty"`
	Members    []string `json:"members"`
}

func (req assignReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.MemberKind == "" {
		return apiutil.ErrMissingMemberKind
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type unassignReq struct {
	token      string
	groupID    string
	Relation   string   `json:"relation,omitempty"`
	MemberKind string   `json:"member_kind,omitempty"`
	Members    []string `json:"members"`
}

func (req unassignReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.MemberKind == "" {
		return apiutil.ErrMissingMemberKind
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type listMembersReq struct {
	token      string
	groupID    string
	permission string
	memberKind string
}

func (req listMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.memberKind == "" {
		return apiutil.ErrMissingMemberKind
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}
