package auth

import (
	"context"
	"errors"
	"time"
)

const MaxLevel = uint64(5)
const MinLevel = uint64(1)

var (
	// ErrMaxLevelExceeded malformed entity.
	ErrMaxLevelExceeded = errors.New("level must be less than or equal 5")

	// ErrBadGroupName malformed entity.
	ErrBadGroupName = errors.New("incorrect group name")

	// ErrGroupConflict group conflict.
	ErrGroupConflict = errors.New("group already exists")

	// ErrCreateGroup indicates failure to create group.
	ErrCreateGroup = errors.New("failed to create group")

	// ErrFetchGroups indicates failure to fetch groups.
	ErrFetchGroups = errors.New("failed to fetch groups")

	// ErrUpdateGroup indicates failure to update group.
	ErrUpdateGroup = errors.New("failed to update group")

	// ErrDeleteGroup indicates failure to delete group.
	ErrDeleteGroup = errors.New("failed to delete group")

	// ErrGroupNotFound indicates failure to find group.
	ErrGroupNotFound = errors.New("failed to find group")

	// ErrAssignToGroup indicates failure to assign member to a group.
	ErrAssignToGroup = errors.New("failed to assign member to a group")

	// ErrUnassignFromGroup indicates failure to unassign member from a group.
	ErrUnassignFromGroup = errors.New("failed to unassign member from a group")

	// ErrUnsupportedContentType indicates unacceptable or lack of Content-Type
	ErrUnsupportedContentType = errors.New("unsupported content type")

	// ErrFailedDecode indicates failed to decode request body
	ErrFailedDecode = errors.New("failed to decode request body")

	// ErrMissingParent indicates that parent can't be found
	ErrMissingParent = errors.New("failed to retrieve parent")

	// ErrGroupNotEmpty indicates group is not empty, can't be deleted.
	ErrGroupNotEmpty = errors.New("group is not empty")

	// ErrMemberAlreadyAssigned indicates that members is already assigned.
	ErrMemberAlreadyAssigned = errors.New("member is already assigned")

	// ErrSelectEntity indicates error while reading entity from database
	ErrSelectEntity = errors.New("select entity from db error")
)

type GroupMetadata map[string]interface{}

type Member struct {
	ID   string
	Type string
}

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

type GroupPage struct {
	PageMetadata
	Groups []Group
}

type MemberPage struct {
	PageMetadata
	Members []Member
}

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
}

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
