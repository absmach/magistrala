package groups

import (
	"regexp"

	"github.com/mainflux/mainflux/internal/groups"
	"github.com/mainflux/mainflux/pkg/errors"
)

var groupRegexp = regexp.MustCompile("^[A-Za-z0-9]+[A-Za-z0-9_-]*$")

type createGroupReq struct {
	token       string
	Name        string                 `json:"name,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req createGroupReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}
	if len(req.Name) > maxNameSize || req.Name == "" || !groupRegexp.MatchString(req.Name) {
		return errors.Wrap(groups.ErrMalformedEntity, groups.ErrBadGroupName)
	}

	return nil
}

type updateGroupReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}

	if req.id == "" {
		return groups.ErrMalformedEntity
	}

	if req.ParentID != "" {
		return groups.ErrParentInvariant
	}

	return nil
}

type listGroupsReq struct {
	token    string
	level    uint64
	metadata groups.Metadata
	name     string
	groupID  string
	tree     bool
}

func (req listGroupsReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}

	if req.level < 0 || req.level > 5 {
		return groups.ErrMalformedEntity
	}

	return nil
}

type listMemberGroupReq struct {
	token    string
	offset   uint64
	limit    uint64
	metadata groups.Metadata
	name     string
	groupID  string
	memberID string
	tree     bool
}

func (req listMemberGroupReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}

	if req.groupID == "" && req.memberID == "" {
		return groups.ErrMalformedEntity
	}

	return nil
}

type memberGroupReq struct {
	token    string
	groupID  string
	memberID string
}

func (req memberGroupReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}

	if req.groupID == "" && req.memberID == "" {
		return groups.ErrMalformedEntity
	}

	return nil
}

type groupReq struct {
	token   string
	groupID string
	name    string
}

func (req groupReq) validate() error {
	if req.token == "" {
		return groups.ErrUnauthorizedAccess
	}

	if req.groupID == "" && req.name == "" {
		return groups.ErrMalformedEntity
	}

	return nil
}
