// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/absmach/magistrala/internal/api"
	"github.com/absmach/magistrala/pkg/apiutil"
	mggroups "github.com/absmach/magistrala/pkg/groups"
	"github.com/absmach/magistrala/pkg/policies"
)

type createGroupReq struct {
	mggroups.Group
	domainID string
}

func (req createGroupReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	if len(req.Name) > api.MaxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateGroupReq struct {
	id          string
	domainID    string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
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
	mggroups.Page
	memberKind string
	memberID   string
	domainID   string
	// - `true`  - result is JSON tree representing groups hierarchy,
	// - `false` - result is JSON array of groups.
	tree bool
}

func (req listGroupsReq) validate() error {
	if req.memberKind == "" {
		return apiutil.ErrMissingMemberKind
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	if req.memberKind == policies.ThingsKind && req.memberID == "" {
		return apiutil.ErrMissingID
	}
	if req.Level > mggroups.MaxLevel {
		return apiutil.ErrInvalidLevel
	}
	if req.Limit > api.MaxLimitSize || req.Limit < 1 {
		return apiutil.ErrLimitSize
	}

	return nil
}

type groupReq struct {
	domainID string
	id string
}

func (req groupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type groupPermsReq struct {
	id       string
	domainID string
}

func (req groupPermsReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}

	return nil
}

type changeGroupStatusReq struct {
	id       string
	domainID string
}

func (req changeGroupStatusReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	return nil
}

type assignReq struct {
	groupID    string
	domainID   string
	Relation   string   `json:"relation,omitempty"`
	MemberKind string   `json:"member_kind,omitempty"`
	Members    []string `json:"members"`
}

func (req assignReq) validate() error {
	if req.domainID == "" {
		return apiutil.ErrMissingID
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
	groupID    string
	domainID   string
	Relation   string   `json:"relation,omitempty"`
	MemberKind string   `json:"member_kind,omitempty"`
	Members    []string `json:"members"`
}

func (req unassignReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.domainID == "" {
		return apiutil.ErrMissingID
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
	groupID    string
	permission string
	memberKind string
	domainID   string
}

func (req listMembersReq) validate() error {
	if req.memberKind == "" {
		return apiutil.ErrMissingMemberKind
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}
	if req.domainID == "" {
		return apiutil.ErrMissingDomainID
	}
	return nil
}
