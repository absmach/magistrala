package groups

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mainflux/mainflux"
)

var (
	_ mainflux.Response = (*memberPageRes)(nil)
	_ mainflux.Response = (*groupRes)(nil)
	_ mainflux.Response = (*groupDeleteRes)(nil)
	_ mainflux.Response = (*assignMemberToGroupRes)(nil)
	_ mainflux.Response = (*removeMemberFromGroupRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Name   string `json:"name"`
}

type memberPageRes struct {
	pageRes
	Members []interface{}
}

func (res memberPageRes) Code() int {
	return http.StatusOK
}

func (res memberPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res memberPageRes) Empty() bool {
	return false
}

type viewGroupRes struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name,omitempty"`
	ParentID    string                 `json:"parent_id,omitempty"`
	OwnerID     string                 `json:"owner_id,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	// Indicates a level in tree hierarchy from first group node.
	Level int `json:"level,omitempty"`
	// Path is a path in a tree, consisted of group names
	// parentName.childrenName1.childrenName2 .
	Path      string          `json:"path"`
	Children  []*viewGroupRes `json:"children"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (res viewGroupRes) Code() int {
	return http.StatusOK
}

func (res viewGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewGroupRes) Empty() bool {
	return false
}

type groupRes struct {
	id      string
	created bool
}

func (res groupRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res groupRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/groups/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res groupRes) Empty() bool {
	return true
}

type groupPageRes struct {
	pageRes
	Groups []viewGroupRes `json:"groups"`
}

func (res groupPageRes) Code() int {
	return http.StatusOK
}

func (res groupPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupPageRes) Empty() bool {
	return false
}

type groupDeleteRes struct{}

func (res groupDeleteRes) Code() int {
	return http.StatusNoContent
}

func (res groupDeleteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupDeleteRes) Empty() bool {
	return true
}

type assignMemberToGroupRes struct{}

func (res assignMemberToGroupRes) Code() int {
	return http.StatusNoContent
}

func (res assignMemberToGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignMemberToGroupRes) Empty() bool {
	return true
}

type removeMemberFromGroupRes struct{}

func (res removeMemberFromGroupRes) Code() int {
	return http.StatusNoContent
}

func (res removeMemberFromGroupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeMemberFromGroupRes) Empty() bool {
	return true
}
