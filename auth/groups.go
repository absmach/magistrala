package auth

import (
	"context"
	"errors"
	"time"
)

const (
	// MaxLevel represents the maximum group hierarchy level.
	MaxLevel = uint64(5)
	// MinLevel represents the minimum group hierarchy level.
	MinLevel = uint64(1)
)

var (
	// ErrAssignToGroup indicates failure to assign member to a group.
	ErrAssignToGroup = errors.New("failed to assign member to a group")

	// ErrUnassignFromGroup indicates failure to unassign member from a group.
	ErrUnassignFromGroup = errors.New("failed to unassign member from a group")

	// ErrMissingParent indicates that parent can't be found
	ErrMissingParent = errors.New("failed to retrieve parent")

	// ErrGroupNotEmpty indicates group is not empty, can't be deleted.
	ErrGroupNotEmpty = errors.New("group is not empty")

	// ErrMemberAlreadyAssigned indicates that members is already assigned.
	ErrMemberAlreadyAssigned = errors.New("member is already assigned")
)

// GroupMetadata defines the Metadata type.
type GroupMetadata map[string]interface{}

// Member represents the member information.
type Member struct {
	ID   string
	Type string
}

// Group represents the group information.
type Group struct {
	ID          string
	OwnerID     string
	ParentID    string
	Name        string
	Description string
	Metadata    GroupMetadata
	// Indicates a level in tree hierarchy.
	// Root node is level 1.
	Level int
	// Path in a tree consisting of group ids
	// parentID1.parentID2.childID1
	// e.g. 01EXPM5Z8HRGFAEWTETR1X1441.01EXPKW2TVK74S5NWQ979VJ4PJ.01EXPKW2TVK74S5NWQ979VJ4PJ
	Path      string
	Children  []*Group
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Size     uint64
	Level    uint64
	Name     string
	Type     string
	Metadata GroupMetadata
}

// GroupPage contains page related metadata as well as list of groups that
// belong to this page.
type GroupPage struct {
	PageMetadata
	Groups []Group
}

// MemberPage contains page related metadata as well as list of members that
// belong to this page.
type MemberPage struct {
	PageMetadata
	Members []Member
}

// GroupService specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type GroupService interface {
	// CreateGroup creates new  group.
	CreateGroup(ctx context.Context, token string, g Group) (Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token string, pm PageMetadata) (GroupPage, error)

	// ListChildren retrieves groups that are children to group identified by parentID
	ListChildren(ctx context.Context, token, parentID string, pm PageMetadata) (GroupPage, error)

	// ListParents retrieves groups that are parent to group identified by childID.
	ListParents(ctx context.Context, token, childID string, pm PageMetadata) (GroupPage, error)

	// ListMembers retrieves everything that is assigned to a group identified by groupID.
	ListMembers(ctx context.Context, token, groupID, groupType string, pm PageMetadata) (MemberPage, error)

	// ListMemberships retrieves all groups for member that is identified with memberID belongs to.
	ListMemberships(ctx context.Context, token, memberID string, pm PageMetadata) (GroupPage, error)

	// RemoveGroup removes the group identified with the provided ID.
	RemoveGroup(ctx context.Context, token, id string) error

	// Assign adds a member with memberID into the group identified by groupID.
	Assign(ctx context.Context, token, groupID, groupType string, memberIDs ...string) error

	// Unassign removes member with memberID from group identified by groupID.
	Unassign(ctx context.Context, token, groupID string, memberIDs ...string) error

	// AssignGroupAccessRights adds access rights on thing groups to user group.
	AssignGroupAccessRights(ctx context.Context, token, thingGroupID, userGroupID string) error
}

// GroupRepository specifies a group persistence API.
type GroupRepository interface {
	// Save group
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group
	Update(ctx context.Context, g Group) (Group, error)

	// Delete a group
	Delete(ctx context.Context, id string) error

	// RetrieveByID retrieves group by its id
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, pm PageMetadata) (GroupPage, error)

	// RetrieveAllParents retrieves all groups that are ancestors to the group with given groupID.
	RetrieveAllParents(ctx context.Context, groupID string, pm PageMetadata) (GroupPage, error)

	// RetrieveAllChildren retrieves all children from group with given groupID up to the hierarchy level.
	RetrieveAllChildren(ctx context.Context, groupID string, pm PageMetadata) (GroupPage, error)

	//  Retrieves list of groups that member belongs to
	Memberships(ctx context.Context, memberID string, pm PageMetadata) (GroupPage, error)

	// Members retrieves everything that is assigned to a group identified by groupID.
	Members(ctx context.Context, groupID, groupType string, pm PageMetadata) (MemberPage, error)

	// Assign adds a member to group.
	Assign(ctx context.Context, groupID, groupType string, memberIDs ...string) error

	// Unassign removes a member from a group
	Unassign(ctx context.Context, groupID string, memberIDs ...string) error
}
